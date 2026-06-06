---
weight: 703
title: "goroutine 调度时机有哪些"
slug: /when
---

在下面这些情形下，goroutine 可能会发生调度，但也并不一定会发生，只是说 Go scheduler 有机会进行调度。

|情形|说明|
|---|---|
|使用关键字 `go`|go 创建一个新的 goroutine，Go scheduler 会考虑调度|
|GC| 由于进行 GC 的 goroutine 也需要在 M 上运行，因此肯定会发生调度。当然，Go scheduler 还会做很多其他的调度，例如调度不涉及堆访问的 goroutine 来运行。GC 不管栈上的内存，只会回收堆上的内存|
|系统调用|当 goroutine 进行系统调用时，会阻塞 M，所以它会被调度走，同时一个新的 goroutine 会被调度上来|
|内存同步访问|atomic，mutex，channel 操作等会使 goroutine 阻塞，因此会被调度走。等条件满足后（例如其他 goroutine 解锁了）还会被调度上来继续运行|
|函数调用|这是“协作式抢占”的实现基础。编译器会在大部分函数的开头插入一段栈检查代码（function prologue），用于判断是否需要扩栈。后台监控线程 `sysmon` 在发现某个 goroutine 运行时间过长时，会把它的栈边界 `stackguard0` 设置为一个特殊值 `stackPreempt`，于是该 goroutine 在下一次函数调用进入 prologue 检查时就会“误以为”栈空间不足，从而跳转进入 runtime，进而触发调度。需要注意，这种方式无法抢占没有函数调用的紧凑循环；为此 Go 1.14 引入了基于信号的“异步抢占”，可以在函数调用之外的位置打断 goroutine|