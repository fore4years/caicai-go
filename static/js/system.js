// ============================================================================
// 原始代码（保留备用，如需回退请取消注释）
// ============================================================================
// document.write("<script type='text/javascript' language=javascript src=" + "https://unpkg.com/axios/dist/axios.min.js" + "></script>");
// // document.write("<script src=\"https://cdn.bootcdn.net/ajax/libs/vue/2.6.13/vue.js\"></script>");
// document.write("<script src=\"/js/vue.js\"></script>");
//
//
// document.addEventListener("DOMContentLoaded", function () {
//
//     // var axios = window.axios = require('axios');
//
//     // 添加请求拦截器
//     axios.interceptors.request.use(function (config) {
//         // 在发送请求之前做些什么
//         return config;
//     }, function (error) {
//         // 对请求错误做些什么
//         return Promise.reject(error);
//     });
//
//     // 添加响应拦截器
//     axios.interceptors.response.use(function (response) {
//         // 2xx 范围内的状态码都会触发该函数。
//         // 对响应数据做点什么
//         if (response.data.code) {
//             if (response.data.code === 200) return response.data.data
//             else return Promise.reject(response.data.message);
//         } else return response;
//     }, function (error) {
//         // 超出 2xx 范围的状态码都会触发该函数。
//         // 对响应错误做点什么
//         return Promise.reject(error);
//     });
// });

// ============================================================================
// 新代码：全局设置 axios baseURL，强制所有请求走当前页面的域名（自带 https://）
// ============================================================================
document.write("<script type='text/javascript' language=javascript src=" + "https://unpkg.com/axios/dist/axios.min.js" + "></script>");
document.write("<script src=\"/js/vue.js\"></script>");

document.addEventListener("DOMContentLoaded", function () {

    // 🌟 强制设置全局 baseURL 为当前页面的域名（自带 https://），
    // 之后 /system/login 会自动拼成 https://test.caiparking.com:444/system/login
    if (window.axios) {
        axios.defaults.baseURL = window.location.origin;
    }

    // 添加请求拦截器
    axios.interceptors.request.use(function (config) {
        // 在发送请求之前做些什么
        return config;
    }, function (error) {
        // 对请求错误做些什么
        return Promise.reject(error);
    });

    // 添加响应拦截器
    axios.interceptors.response.use(function (response) {
        // 2xx 范围内的状态码都会触发该函数。
        // 对响应数据做点什么
        if (response.data.code) {
            if (response.data.code === 200) return response.data.data
            else return Promise.reject(response.data.message);
        } else return response;
    }, function (error) {
        // 超出 2xx 范围的状态码都会触发该函数。
        // 对响应错误做点什么
        return Promise.reject(error);
    });
});
