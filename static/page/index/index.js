let left = new Vue({
    el: '#left',
    data: {
        select: ['item', 'item', 'item', 'item', 'item', 'item', 'item', 'item', 'item', 'item', 'item', 'item', 'item', 'item'],
        jiantou: ['jiantou', 'jiantou'],
        chesuo_c: ['chesuo_c', 'chesuo_c', 'chesuo_c'],
        xiaoxi_c: ['xiaoxi_c', 'xiaoxi_c'],
        chesuo_zk: false,
        xiaoxi_zk: false,
        right_url: '../option/option.html'
    },
    methods: {
        select_c(index) {
            for (let i = 0; i < this.select.length; i++) {
                this.select[i] = 'item';
            }
            this.select[index] = 'item active';
            switch (index) {
                case 0:
                    this.right_url = '../option/option.html';
                    break;
                case 1:
                    this.right_url = '../option/place.html';
                    break;
                case 2:
                    if (this.chesuo_zk) {
                        this.chesuo_zk = false;
                        this.jiantou[0] = 'jiantou';
                    } else {
                        this.chesuo_zk = true;
                        this.jiantou[0] = 'jiantou active';
                    }
                    break;
                case 4:
                    this.right_url = '../option/shouyi.html';
                    break;
                case 5:
                    if (this.xiaoxi_zk) {
                        this.xiaoxi_zk = false;
                        this.jiantou[1] = 'jiantou';
                    } else {
                        this.xiaoxi_zk = true;
                        this.jiantou[1] = 'jiantou active';
                    }
                    break;
                case 6:
                    this.right_url = '../option/mima.html';
                    break;
                case 7:
                    this.right_url = '../option/charging_station.html';
                    break;
                case 8:
                    this.right_url = '../option/gateway.html';
                    break;
                case 9:
                    this.right_url = '../option/om_user.html';
                    break;
                case 10:
                    this.right_url = '../option/charging.html';
                    break;
                case 11:
                    this.right_url = '../option/charging_four.html';
                    break;
                case 12:
                    this.right_url = '../option/product.html';
                    break;
                case 13:
                    this.right_url = '../orderManage/orderManage.html';
                    break;
            }
        },
        chesuo_ck(index) {
            for (let i = 0; i < this.chesuo_c.length; i++) {
                this.chesuo_c[i] = 'chesuo_c';
            }
            this.chesuo_c[index] = 'chesuo_c active';
            switch (index) {
                case 0:
                    this.right_url = '../option/lock.html';
                    break;
                case 1:
                    this.right_url = '../option/repair.html';
                    break;
                case 2:
                    this.right_url = '../option/fault.html';
                    break;
            }
        },
        xiaoxi_ck(index) {
            for (let i = 0; i < this.xiaoxi_c.length; i++) {
                this.xiaoxi_c[i] = 'xiaoxi_c';
            }
            this.xiaoxi_c[index] = 'xiaoxi_c active';
            switch (index) {
                case 0:
                    this.right_url = '../option/xiaoxi.html';
                    break;
                case 1:
                    this.right_url = '../option/xiaoxiset.html';
                    break;
            }
        }
    }
});

let title = new Vue({
    el: '#title',
    data: {
        account: ''
    }
}); 