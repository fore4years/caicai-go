package service

import (
	"encoding/hex"
	"net"
	"strconv"
	"sync"

	"caicai-go/logger"
	"caicai-go/protocol"
)

// ============================================================================
// TCP 设备服务器：兼容仍使用自定义二进制协议的老设备（未迁移到 MQTT）。
// 帧格式见 protocol 包：
//   - 4G 上行：[imei 8][AA 02 ctl cmd len_h len_l data crc CD]
//   - LoRa 上行：[nid 4][AA 01 ctl cmd len_h len_l data crc CD]
// 下行：SendCommand 里按 deviceID 是否已建立 TCP 连接决定走 TCP（二进制帧）还是 MQTT（JSON）。
// ============================================================================

var (
	// tcpConns 维护 TCP 设备连接：deviceID（hex 字符串）→ net.Conn。
	tcpConns sync.Map
)

// StartTCPServer 启动 TCP 设备服务器，监听指定端口。
func StartTCPServer(port int) error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	logger.Mylog.Info().Msgf("TCP 设备服务器启动, 监听端口: %d", port)
	go acceptTCPLoop(ln)
	return nil
}

func acceptTCPLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Mylog.Warn().Err(err).Msg("TCP Accept 失败")
			return
		}
		go handleTCPConn(conn)
	}
}

// handleTCPConn 处理单条 TCP 连接：按长度字段切帧，解析并注册设备，缓存上行数据。
func handleTCPConn(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	logger.Mylog.Info().Msgf("TCP 设备连接: %s", remote)

	buf := make([]byte, 4096)
	pending := make([]byte, 0, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		pending = append(pending, buf[:n]...)
		for {
			frame, deviceID, rest, ok := extractTCPFrame(pending)
			if !ok {
				break
			}
			pending = rest
			if deviceID != "" {
				tcpConns.Store(deviceID, conn)
			}
			processDeviceFrame(frame, "tcp:"+remote)
		}
		// 防止无法切帧的脏数据导致缓冲无限增长
		if len(pending) > 4096 {
			logger.Mylog.Warn().Msgf("TCP 设备缓冲溢出，丢弃 %d 字节, remote: %s", len(pending), remote)
			pending = pending[:0]
		}
	}

	// 连接断开：反注册
	tcpConns.Range(func(k, v interface{}) bool {
		if v.(net.Conn) == conn {
			tcpConns.Delete(k)
		}
		return true
	})
	logger.Mylog.Info().Msgf("TCP 设备断开: %s", remote)
}

// extractTCPFrame 从字节流中提取一帧，返回帧字节、设备ID（hex）与剩余字节。
// 4G：b[8]==0xAA && b[9]==0x02，长度字段在 12-13；LoRa：b[4]==0xAA && b[5]==0x01，长度字段在 8-9。
func extractTCPFrame(b []byte) (frame []byte, deviceID string, rest []byte, ok bool) {
	if len(b) < 9 {
		return nil, "", b, false
	}
	// 4G
	if len(b) >= 12 && b[8] == protocol.Header && b[9] == protocol.Version4G {
		dataLen := int(b[12])<<8 | int(b[13])
		total := 12 + 2 + dataLen + 2 // 前置12 + len2 + data + crc + CD
		if len(b) < total {
			return nil, "", b, false
		}
		return b[:total], hex.EncodeToString(b[:8]), b[total:], true
	}
	// LoRa
	if len(b) >= 8 && b[4] == protocol.Header && b[5] == protocol.Version {
		dataLen := int(b[8])<<8 | int(b[9])
		total := 8 + 2 + dataLen + 2 // 前置8 + len2 + data + crc + CD
		if len(b) < total {
			return nil, "", b, false
		}
		return b[:total], hex.EncodeToString(b[:4]), b[total:], true
	}
	return nil, "", b, false
}

// SendTCP 向已建立 TCP 连接的设备下发二进制帧。设备未连接时返回 false。
func SendTCP(deviceID string, frame []byte) bool {
	v, ok := tcpConns.Load(deviceID)
	if !ok {
		return false
	}
	conn := v.(net.Conn)
	if _, err := conn.Write(frame); err != nil {
		logger.Mylog.Err(err).Msgf("TCP 下发失败, device: %s", deviceID)
		return false
	}
	return true
}
