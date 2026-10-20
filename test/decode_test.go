package test

import (
	"caicai-go/protocol"
	"encoding/hex"
	"testing"
)

func TestDecode(t *testing.T) {
	b, err := hex.DecodeString("00000000AA0100FF0006D4AD205FCFEC6BCD")
	if err != nil {
		t.Fatalf("hex 解码失败: %v", err)
	}

	// 2. 调用被测函数
	deviceID, ctl, cmd, data, err := protocol.Decode(b)
	if err != nil {
		t.Fatalf("Decode 返回错误: %v", err)
	}

	// 3. 断言结果
	if deviceID != "00000000" {
		t.Errorf("deviceID = %q, 期望 %q", deviceID, "00000000")
	}
	if ctl != 0x00 {
		t.Errorf("ctl = 0x%02X, 期望 0x00", ctl)
	}
	if cmd != 0xFF {
		t.Errorf("cmd = 0x%02X, 期望 0xFF", cmd)
	}

	wantData, _ := hex.DecodeString("D4AD205FCFEC")
	if string(data) != string(wantData) {
		t.Errorf("data = %X, 期望 %X", data, wantData)
	}

}
