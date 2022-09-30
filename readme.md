## RPC framework

RPC(Remote Procedure Call，远程过程调用)是一种计算机通信协议，
允许调用不同进程空间的程序。RPC 的客户端和服务器可以在一台机器上，
也可以在不同的机器上。程序员使用时，就像调用本地程序一样，无需关注内部的实现细节。

## RPC 框架需要解决什么问题
### 1. 如何解决网络通信
* 传输协议的选择

    一般会选择 TCP 协议或者 HTTP 协议；那如果两个应用程序位于相同的机器，也可以选择 Unix Socket 协议

* 通讯连接建立
    
    RPC 所有交换的数据都在这个连接里传输，这个连接可以是按需连接（需要调用时就先建立连接，调用结束后就立马断掉），也可以是长连接（客户端和服务器建立起连接之后保持长期持有，不管此时有无数据包的发送，可以配合心跳检测机制定期检测建立的连接是否存活有效），多个远程过程调用共享同一个连接。

### 2. 如何进行服务发现
*  从服务端的角度
    
    当服务提供者启动的时候，需要将自己提供的服务注册到指定的注册中心，以便服务消费者能够通过服务注册中心进行查找
* 从调用者的角度

    当服务提供者由于各种原因致使提供的服务停止时，需要向注册中心注销停止的服务；服务的提供者需要定期向服务注册中心发送心跳检测，服务注册中心如果一段时间未收到来自服务提供者的心跳后，认为该服务提供者已经停止服务，则将该服务从注册中心上去掉。

### 3. 如何序列化和反序列化
    最常用的 JSON 或者 XML，那如果报文比较大，还可能会选择 gob、protobuf 等其他的编码方式，甚至编码之后，再进行压缩。接收端获取报文则需要相反的过程，先解压再解码。

再进一步，假设服务端是不同的团队提供的，如果没有统一的 RPC 
框架，各个团队的服务提供方就需要各自实现一套消息编解码、连接池、收发线程、超时处理等“业务之外”的重复技术劳动，造成整体的低效。因此，“业务之外”的这部分公共的能力，即是 RPC 框架所需要具备的能力。