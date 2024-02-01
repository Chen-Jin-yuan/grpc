import requests
import json
import consul
import time

class ConsulClient:
    def __init__(self, host='localhost', port=8500):
        self.consul = consul.Consul(host=host, port=port)

    def register_service(self, name, id, port, ip):
        reg = {
            'name': name,
            'service_id': id,
            'port': port,
            'address': ip
        }

        print(f"Trying to register service [ name: {name}, id: {id}, address: {ip}:{port} ]")
        return self.consul.agent.service.register(**reg)


    def deregister_service(self, id):
        print(f"Trying to deregister service id: {id}")
        return self.consul.agent.service.deregister(id)

    def get_service(self, name):
        _, nodes = self.consul.health.service(service=name, passing=True)
        id_list = []
        ip_list = []
        port_list = []

        if len(nodes) == 0:
            print('service is empty.')

        for node in nodes:
            service = node.get('Service')

            id_list.append(service['ID'])
            ip_list.append(service['Address'])
            port_list.append(service['Port'])

        return id_list, ip_list, port_list



def test_consul():
    # 示例用法
    consul_address = "127.0.0.1"
    service_name = "helloServer"

    consul_client = ConsulClient(consul_address, 8500)

    id_list, ip_list, port_list = consul_client.get_service(service_name)

    consul_client.deregister_service(id=id_list[0])
    time.sleep(1)
    consul_client.register_service(name=service_name, id=id_list[0], port=port_list[0], ip=ip_list[0])

def test():
    print("start server_test")

    # 新的配置映射
    new_config_map = {
        "helloServer": [
            {"IP": "127.0.0.1:50000", "Weight": 0.8},
            {"IP": "127.0.0.1:50001", "Weight": 0.2}
        ],
        "service2": [
            {"IP": "20.0.0.1", "Weight": 0.5},
            {"IP": "20.0.0.2", "Weight": 0.5}
        ]
    }

    # API 地址
    api_url = "http://localhost:10001/updateConfigMap"  # 将 your_port 替换为你的实际端口号

    # 发送 POST 请求
    response = requests.post(api_url, json=new_config_map)

    # 检查响应
    if response.status_code == 200:
        print("ConfigMap updated successfully")
    else:
        print(f"Error updating ConfigMap. Status code: {response.status_code}")
        print(response.text)


# test()
test_consul()


