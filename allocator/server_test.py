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


# 目标 URL
url = "http://localhost:10001/modify-group"

# 准备要发送的数据
modify_group_number = {
    "helloServer": [2, 2, 3]
}

modify_config(url, modify_group_number)