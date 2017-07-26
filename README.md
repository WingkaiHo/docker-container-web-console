Docker 容器web控制台
---------------

## 1. 代码来源

    https://gist.github.com/Humerus/0268c62f359f7ee1ee2d

    修改当前web console 关闭以后， bash 没有关闭的问题


## 2. 使用说明

### 2.1 服务部署说明
    代码就是Docker web控制台的web服务器。需要把jss以及html文件存放在一个文件夹里面。服务端口默认是8080， 或者可以通过参数`-port=`

### 2.2 服务和docker通信配置
    服务器目前`-host=tcp:ip:port`模式， 需要配置docker打开端口

```
vim /etc/systemd/system/multi-user.target.wants/docker.service 
-H tcp://0.0.0.0:2375 -H unix:///var/run/docker.sock 
```

### 2.3 启动服务

   启动服务[通过ip端口和docker通信]:
```
./docker-container-web-console -port=8080 -host=tcp:127.0.0.1:2375
```

  启动服务[通过本地socker端口和docker通信]:

```
./docker-container-web-console -port=8080 -host=unix:/var/run/docker.sock
```

   通过浏览器进入docker容器<container-id>必须全称

```
http://127.0.0.1:8080?id=<container-id>
```


### 2.4 Docker 启动

  通过脚本build-docker-image.sh, 生成docker镜像

  启动
```
  docker run -d  --name docker-container-web -p 2378:2378 -v /var/run/docker.sock:/var/run/docker.sock docker-web-console:v1.0
``` 
