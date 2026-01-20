# 压力测试工具

### 简介

力求简化压力测试配置，一个简单的yaml，即可启动测试，支持云原生

### 使用说明

#### 配置文件

```yaml
log:
  path: logs
result:
  storageType: file
```

#### 测试构建

```yaml
- name: f_stage
  number_concurrent: 100
  sampling: 1
  ramp_up: 10
  duration: 10s
  execution_mode: random
  tasks:
    - type: grpc
      task:
        name: gg_task
        description: 登录
        url: localhost:50051
        service: Greeter
        method: SayHello
        params:
          - type: map
            param: { "name": "weicai" }
```

### 构建

#### docker

快速构建版：
`docker build -t wtester:v1 .`

日志输出版：
`docker build -t wtester:v1 . --progress=plain --no-cache`

### 领域模型

#### stage

测试阶段，测试阶段之间串行执行测试，每个测试阶段包含多个测试任务。测试阶段用来配置测试任务的执行方案：日志采样、并发启动、执行方式等

#### task

测试任务，实际执行的测试任务，可以配置此任务要执行的request等信息。目前支持：http(s)，grpc(利用grpc反射)

#### parameter

http等请求需要的参数

### 测试方案

```yaml
- name: f_stage
  number_concurrent: 100
  sampling: 1
  ramp_up: 10
  duration: 10s
  execution_mode: random
  tasks:
    - type: grpc
      task:
        name: gg_task
        description: 登录
        url: 127.0.0.1:50051
        service: Greeter
        method: SayHello
        params:
          - type: map
            param: { "username": "weicai" }
          - type: sequence_map
            param:
              [
                { "password": "www", "number": 100 },
                { "password": "xxn", "number": 100 },
                { "password": "lo", "number": 100 },
              ]
          - type: file
            param: D:\GoProject\wtester\configs\filepara.json
          - type: sequence
            param:
              sequences:
                - start: 1
                  end: 3
                  step: 1
                  type: int
                  key: username
    - type: http
      task:
        name: ff_task
        description: 登录
        url: http://127.0.0.1:31704/report/temp
        params:
          - type: map
            param: { "username": "weicai" }
          - type: sequence_map
            param:
              [
                { "password": "www", "number": 100 },
                { "password": "xxn", "number": 100 },
                { "password": "lo", "number": 100 },
              ]
          - type: file
            param: D:\GoProject\wtester\configs\filepara.json
          - type: sequence
            param:
              sequences:
                - start: 1
                  end: 3
                  step: 1
                  type: int
                  key: username
```