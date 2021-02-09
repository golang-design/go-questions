---
weight: 405
title: "关闭一个 channel 的过程是怎样的"
slug: /close
---

关闭某个 channel，会执行函数 `closechan`：

```golang
func closechan(c *hchan) {
	// 关闭一个 nil channel，panic
	if c == nil {
		panic(plainError("close of nil channel"))
	}

	// 上锁
	lock(&c.lock)
	// 如果 channel 已经关闭
	if c.closed != 0 {
		unlock(&c.lock)
		// panic
		panic(plainError("close of closed channel"))
	}

	// …………

	// 修改关闭状态
	c.closed = 1

	var glist *g

	// 将 channel 所有等待接收队列的里 sudog 释放
	for {
		// 从接收队列里出队一个 sudog
		sg := c.recvq.dequeue()
		// 出队完毕，跳出循环
		if sg == nil {
			break
		}

		// 如果 elem 不为空，说明此 receiver 未忽略接收数据
		// 给它赋一个相应类型的零值
		if sg.elem != nil {
			typedmemclr(c.elemtype, sg.elem)
			sg.elem = nil
		}
		if sg.releasetime != 0 {
			sg.releasetime = cputicks()
		}
		// 取出 goroutine
		gp := sg.g
		gp.param = nil
		if raceenabled {
			raceacquireg(gp, unsafe.Pointer(c))
		}
		// 相连，形成链表
		gp.schedlink.set(glist)
		glist = gp
	}

	// 将 channel 等待发送队列里的 sudog 释放
	// 如果存在，这些 goroutine 将会 panic
	for {
		// 从发送队列里出队一个 sudog
		sg := c.sendq.dequeue()
		if sg == nil {
			break
		}

		// 发送者会 panic
		sg.elem = nil
		if sg.releasetime != 0 {
			sg.releasetime = cputicks()
		}
		gp := sg.g
		gp.param = nil
		if raceenabled {
			raceacquireg(gp, unsafe.Pointer(c))
		}
		// 形成链表
		gp.schedlink.set(glist)
		glist = gp
	}
	// 解锁
	unlock(&c.lock)

	// Ready all Gs now that we've dropped the channel lock.
	// 遍历链表
	for glist != nil {
		// 取最后一个
		gp := glist
		// 向前走一步，下一个唤醒的 g
		glist = glist.schedlink.ptr()
		gp.schedlink = 0
		// 唤醒相应 goroutine
		goready(gp, 3)
	}
}
```

close 逻辑比较简单，对于一个 channel，recvq 和 sendq 中分别保存了阻塞的发送者和接收者。关闭 channel 后，对于等待接收者而言，会收到一个相应类型的零值。对于等待发送者，会直接 panic。所以，在不了解 channel 还有没有接收者的情况下，不能贸然关闭 channel。

close 函数先上一把大锁，接着把所有挂在这个 channel 上的 sender 和 receiver 全都连成一个 sudog 链表，再解锁。最后，再将所有的 sudog 全都唤醒。

唤醒之后，该干嘛干嘛。sender 会继续执行 chansend 函数里 goparkunlock 函数之后的代码，很不幸，检测到 channel 已经关闭了，panic。receiver 则比较幸运，进行一些扫尾工作后，返回。这里，selected 返回 true，而返回值 received 则要根据 channel 是否关闭，返回不同的值。如果 channel 关闭，received 为 false，否则为 true。这我们分析的这种情况下，received 返回 false。
