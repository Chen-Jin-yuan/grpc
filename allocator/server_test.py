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

def modify_block_len(url, block_len):
    # 将数据编码为 JSON 格式
    json_data = json.dumps(block_len)

    # 设置请求头
    headers = {
        "Content-Type": "application/json"
    }

    # 发送 POST 请求
    response = requests.post(url, data=json_data, headers=headers)

    # 打印响应结果
    print(response.status_code)

# 目标 URL
url = "http://localhost:10001/modify-block-len"

# 准备要发送的数据
modify_group_number = {
    "helloServer": [2, 2, 3]
}
print("start")
modify_block_len(url, 4)