# grpc
> 造点轮子
> 
allocator 对外暴露三个 API：
1. 获取计数（排队数、将发送请求数）
2. 获取下游微服务的分组、配置信息
3. 修改配置 config 的接口

三个接口使用方式如下（对应三个函数）：
```python 
import requests
import json

def modify_config(url, modify_group):
    # 将数据编码为 JSON 格式
    json_data = json.dumps(modify_group)

    # 设置请求头
    headers = {
        "Content-Type": "application/json"
    }

    # 发送 POST 请求
    response = requests.post(url, data=json_data, headers=headers)

    # 打印响应结果
    print(response.status_code)
    print(response.text)

def get_svc_config(url, svc_name):
    # 设置 GET 请求的参数
    params = {"name": svc_name}

    # 发送带参数的 GET 请求
    response = requests.get(url, params=params)

    # 打印响应结果
    print(response.status_code)
    print(response.text)

def get_counter(url):
    response = requests.get(url)

    # 打印响应结果
    print(response.status_code)

    json_data = response.json()

    print(json_data["waiting_requests"])
    print()
    print(json_data["out_ready_requests"])


# 1.获取计数情况
url_c = "http://10.244.68.158:10001/counter"
get_counter(url_c)

# 2.获取user服务的配置情况
url_s = "http://10.244.68.158:10001/svc-info"
get_svc_config(url_s, "srv-user")

# 3.修改config配置，比如将profile修改为3个组，每个组副本数是2：2：3
urlm = "http://10.244.68.158:10001/modify-group"
modify_group_number = {
    "srv-profile": [2, 2, 3]
}
modify_config(urlm, modify_group_number)
```