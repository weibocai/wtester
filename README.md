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