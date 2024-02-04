```
env: 
​    1 master and 2 work nodes
​    k8s version: 1.26.0
​    Container Runtime: containerd
​    Runtime Version: 1.7.0
​    Ubuntu 22.04
​    cgroup version: v1!!!(very important)
    go version: go1.19.1 linux/amd64
```

Modified kubelet source code: https://github.com/Jiaxuan-C/kubernetes-1.26.0-migration/tree/master

Modified containerd source code: https://github.com/Jiaxuan-C/containerd-1.7.0-migration/tree/master

![image](https://github.com/Jiaxuan-C/migration-operator/blob/main/demo.gif)
