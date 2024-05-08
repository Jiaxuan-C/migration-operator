# 〇、声明
**如果您仅想快速了解本项目在干什么，请参考*第二部分*内容并观看视频即可(3min钟短视频)**

https://www.bilibili.com/video/BV1nA4m1P7w1/?vd_source=f5d12b3894e2c1a98b31ea20b7fb0c88

**如果您想进一步了解本项目的架构、思想以及改进请阅读下文**
* [一、前言](#一、前言(这不重要，可以直接看第四部分的效果演示))
* [二、基本功能介绍](#二、基本功能介绍)
* [三、效果演示](#三、效果演示)
* [四、快速启动](#四、快速启动(其实麻烦的，建议叫龟速启动))
* [五、架构设计及原理概要](#五、架构设计及原理概要)
* [六、目前待优化的点、未来此方向前景展望](#六、目前待优化的点、未来此方向前景展望)
  

# 一、前言

Kubernetes平台对于无状态的应用有着很完善的管理能力，但对于有状态应用的维护仍是其薄弱之处(即使它拥有强劲的StatefulSet控制器)。当集群中的物理节点出现宕机、需要维护或因资源紧缺而驱逐Pod时，这些Pod往往会在其它节点上重启，丢失原有状态，这对于需要长期运行且有状态的工作负载是十分不利的，如HPC(High performance computing)类，最糟糕的结果是完全丢失几个小时、几天的计算数据。对此，最好是在物理节点发生意外前感知、并迁移这些有状态的应用，但一直以来Kubernetes并没有支持该项功能，直到2023年1月底，Kubernetes社区接受了一项容器Checkpoint的提案，截止至今，最新版本的Kubernetes中已支持相关功能的测试版本，但Kubernetes尚未给出容器Restore的方案。与此同时也有很多开发者对Kubernetes进行二次开发，以支持其个性化的Pod的迁移需求。此外，在2022年发布的1.24版本的Kubernetes中宣布正式弃用Dockershim，使用Containerd作为其默认的容器运行时，所以本项目首选关注Kubernetes和Containerd的集成。

目前官方开源社区现在有一个Redhat的团队，叫Adrian Reber的大佬带队一直在致力于给k8s添加checkpoint的功能，但仅是checkpoint。那对于迁移来讲，即要有Checkpoint 还要有Restore，何况我们要求的是**热迁移**；对Adrian Reber项目感兴趣的可以去看这个issue：https://github.com/kubernetes/enhancements/issues/2008 ，或者去Google搜这个大佬，可以看到他们的技术分享。

# 二、基本功能介绍
本项目在k8s平台上，基于Pre-Copy技术，实现了对**有状态**的Pod的**热迁移**；为了进一步降低被迁移应用的**停机时间**，本项目还对容器的文件系统同步过程做了些文章，后面有详细介绍。

目前网络资源中有朋友**混淆热迁移的概念**，热迁移的停机时间(Downtime)是远小于冷迁移的。我曾在网上看到这样的贴子“基于Docker实现的热迁移方案”，里面介绍了使用docker的Checkpoint和Restore命令实现的有状态容器迁移，但很显然，**简单的C/R仅仅是冷迁移**，这根本不叫热迁移。

​好了，到这儿来说懂的估计也知道我在干嘛了，不懂的朋友不用急，**请直接看效果演示**，看完基本就知道我在干嘛了。

其实我之前在微信的黄大年公众号看到了华为提的云计算项目需求之一就有这个Pod热迁移的项目，好像还搞了个什么“招榜悬赏”的活动，当时想搞，奈何没有团队，实验室组里也只有我自己在搞这个迁移方向。。。

# 三、效果演示

朋友们，给你们两个选择：

1. 看我这一大坨GIF，它就在下面。 
2. **看我幽默、风趣、详细、激情......的b站视频（没事，不用给视频点赞）**：https://www.bilibili.com/video/BV1nA4m1P7w1/?vd_source=f5d12b3894e2c1a98b31ea20b7fb0c88

![Demo](demo.gif)

# 四、快速启动(其实很麻烦，建议叫龟速启动)

## 1. 基本环境

```bash
env:
	1 master and 2 work nodes
	k8s version: 1.26.0
	Container Runtime: containerd
	Runtime Version: 1.7.0
	Ubuntu 22.04
	cgroup version: v1!!!
	go version: go1.19.1 linux/amd64
```

注意了，这有个大坑，当时卡了我一个月，cgroup version一定要使用v1！还好当时在Github上联系到了 Adrian Reber大佬，当时我发现自己的项目出现问题，然后就去复现 Adrian Reber团队的方案，结果他们的方案和我项目出现了相同的bug。。。此处附上当时的对话和一些问题细节，具体的对话可以到https://github.com/kubernetes/enhancements/issues/2008 找，不过太久了，估计被hidden了，需要手动展开（当时我Github就叫qiubabimunieniu，后来需要在论文中公开github地址，我就改名了。。。）：

![image-20240324201105420](README.assets/image-20240324204629679.png)


除了准备上述Kubernetes集群环境外，你还需要自己编译我修改过的Kubelet和Containerd的源码：

## 2. Kubelet

```
代码在这：https://github.com/Jiaxuan-C/kubernetes-1.26.0-migration/tree/master
```

​把它下载下来，只编译kubelet就行，不然巨慢。之前我的K8s中的kubelet是systemd管理的，所以需要你在work node上停掉你之前的kubelet服务，启动我们编译好的。

## 3. Containerd

```
代码在这：https://github.com/Jiaxuan-C/containerd-1.7.0-migration/tree/master
```

​额，把它下载下来，这个全编译，然后编译好的文件全丢到/user/local/bin，当然前提是你的containerd之前的环境变量就配置在这。你也可以像kubelet那样，不过我喜欢这么搞，毕竟它编译好了会出来一大坨bin文件。

**好了你安装完了，去试试效果吧，具体怎么操作参考第三部分。**

# 五、架构设计及原理概要（涉及学术，暂时隐藏，成果公开后再次开放）

# 六、目前待优化的点、未来此方向前景展望（涉及学术，暂时隐藏，成果公开后再次开放）
