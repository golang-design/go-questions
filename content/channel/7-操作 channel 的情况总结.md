---
weight: 407
title: "操作 channel 的情况总结"
slug: /ops
---

总结一下操作 channel 的结果：

|操作|nil channel|closed channel|not nil, not closed channel|
|---|---|---|---|
|close|panic|panic|正常关闭|
|读 <- ch|阻塞|读到对应类型的零值|阻塞或正常读取数据。缓冲型 channel 为空或非缓冲型 channel 没有等待发送者时会阻塞|
|写 ch <-|阻塞|panic|阻塞或正常写入数据。非缓冲型 channel 没有等待接收者或缓冲型 channel buf 满时会被阻塞|

总结一下，发生 panic 的情况有三种：向一个关闭的 channel 进行写操作；关闭一个 nil 的 channel；重复关闭一个 channel。

读、写一个 nil channel 都会被阻塞。
