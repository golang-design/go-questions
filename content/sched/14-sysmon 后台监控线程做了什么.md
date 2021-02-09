---
weight: 714
title: "sysmon 后台监控线程做了什么"
slug: /sysmon
---

在 `runtime.main()` 函数中，执行 `runtime_init()` 前，会启动一个 sysmon 的监控线程，执行后台监控任务：

```golang
systemstack(func() {
	// 创建监控线程，该线程独立于调度器，不需要跟 p 关联即可运行
	newm(sysmon, nil)
})
```

`sysmon` 函数不依赖 P 直接执行，通过 newm 函数创建一个工作线程：

```golang
func newm(fn func(), _p_ *p) {
	// 创建 m 对象
	mp := allocm(_p_, fn)
	// 暂存 m
	mp.nextp.set(_p_)
	mp.sigmask = initSigmask
	
	// ……………………
	
	execLock.rlock() // Prevent process clone.
	// 创建系统线程
	newosproc(mp, unsafe.Pointer(mp.g0.stack.hi))
	execLock.runlock()
}
```

先调用 `allocm` 在堆上创建一个 m，接着调用 `newosproc` 函数启动一个工作线程：

```golang
// src/runtime/os_linux.go
//go:nowritebarrier
func newosproc(mp *m, stk unsafe.Pointer) {
	// ……………………

	ret := clone(cloneFlags, stk, unsafe.Pointer(mp), unsafe.Pointer(mp.g0), unsafe.Pointer(funcPC(mstart)))

	// ……………………
}
```

核心就是调用 clone 函数创建系统线程，新线程从 mstart 函数开始执行。`clone` 函数由汇编语言实现：

```golang
// int32 clone(int32 flags, void *stk, M *mp, G *gp, void (*fn)(void));
TEXT runtime·clone(SB),NOSPLIT,$0
    // 准备系统调用的参数
    MOVL	flags+0(FP), DI
    MOVQ	stk+8(FP), SI
    MOVQ	$0, DX
	MOVQ	$0, R10

	// 将 mp，gp，fn 拷贝到寄存器，对子线程可见
	MOVQ	mp+16(FP), R8
	MOVQ	gp+24(FP), R9
	MOVQ	fn+32(FP), R12

    // 系统调用 clone
	MOVL	$56, AX
	SYSCALL

	// In parent, return.
	CMPQ	AX, $0
	JEQ	3(PC)
	// 父线程，返回
	MOVL	AX, ret+40(FP)
	RET

	// In child, on new stack.
	// 在子线程中。设置 CPU 栈顶寄存器指向子线程的栈顶
	MOVQ	SI, SP

	// If g or m are nil, skip Go-related setup.
	CMPQ	R8, $0    // m
    JEQ	nog
    CMPQ	R9, $0    // g
	JEQ	nog

	// Initialize m->procid to Linux tid
	// 通过 gettid 系统调用获取线程 ID（tid）
	MOVL	$186, AX	// gettid
	SYSCALL
	// 设置 m.procid = tid
	MOVQ	AX, m_procid(R8)

	// Set FS to point at m->tls.
	// 新线程刚刚创建出来，还未设置线程本地存储，即 m 结构体对象还未与工作线程关联起来，
    // 下面的指令负责设置新线程的 TLS，把 m 对象和工作线程关联起来
	LEAQ	m_tls(R8), DI
	CALL	runtime·settls(SB)

	// In child, set up new stack
	get_tls(CX)
	MOVQ	R8, g_m(R9) // g.m = m
	MOVQ	R9, g(CX) // tls.g = &m.g0
	CALL	runtime·stackcheck(SB)

nog:
	// Call fn
	// 调用 mstart 函数。永不返回
	CALL	R12

	// It shouldn't return. If it does, exit that thread.
	MOVL	$111, DI
    MOVL	$60, AX
	SYSCALL
	JMP	-3(PC)	// keep exiting
```

先是为 clone 系统调用准备参数，参数通过寄存器传递。第一个参数指定内核创建线程时的选项，第二个参数指定新线程应该使用的栈，这两个参数都是通过 newosproc 函数传递进来的。

接着将 m, g0, fn 分别保存到寄存器中，待子线程创建好后再拿出来使用。因为这些参数此时是在父线程的栈上，若不保存到寄存器中，子线程就取不出来了。

> 这个几个参数保存在父线程的寄存器中，创建子线程时，操作系统内核会把父线程所有的寄存器帮我们复制一份给子线程，所以当子线程开始运行时就能拿到父线程保存在寄存器中的值，从而拿到这几个参数。

之后，调用 clone 系统调用，内核帮我们创建出了一个子线程。相当于原来的一个执行分支现在变成了两个执行分支，于是会有两个返回。这和著名的 fork 系统调用类似，根据返回值来判断现在是处于父线程还是子线程。

如果是父线程，就直接返回了。如果是子线程，接着还要执行一堆操作，例如设置 tls，设置 m.procid 等等。

最后执行 mstart 函数，这是在 newosproc 函数传递进来的。`mstart` 函数再调用 `mstart1`，在 `mstart1` 里会执行这一行：

```golang
// 执行启动函数。初始化过程中，fn == nil
if fn := _g_.m.mstartfn; fn != nil {
	fn()
}
```

之前我们在讲初始化的时候，这里的 fn 是空，会跳过的。但在这里，fn 就是最开始在 `runtime.main` 里设置的 `sysmon` 函数，因此这里会执行 `sysmon`，而它又是一个无限循环，永不返回。

所以，这里不会执行到 mstart1 函数后面的 schedule 函数，也就不会进入 schedule 循环。因此这是一个不用和 p 结合的 m，它直接在后台执行，默默地执行监控任务。

接下来，我们就来看 `sysmon` 函数到底做了什么？

> `sysmon` 执行一个无限循环，一开始每次循环休眠 20us，之后（1 ms 后）每次休眠时间倍增，最终每一轮都会休眠 10ms。

> `sysmon` 中会进行 netpool（获取 fd 事件）、retake（抢占）、forcegc（按时间强制执行 gc），scavenge heap（释放自由列表中多余的项减少内存占用）等处理。

和调度相关的，我们只关心 retake 函数：

```golang
func retake(now int64) uint32 {
	n := 0
	// 遍历所有的 p
	for i := int32(0); i < gomaxprocs; i++ {
		_p_ := allp[i]
		if _p_ == nil {
			continue
		}
		// 用于 sysmon 线程记录被监控 p 的系统调用时间和运行时间
		pd := &_p_.sysmontick
		// p 的状态
		s := _p_.status
		if s == _Psyscall {
			// P 处于系统调用之中，需要检查是否需要抢占
			// Retake P from syscall if it's there for more than 1 sysmon tick (at least 20us).
			// _p_.syscalltick 用于记录系统调用的次数，在完成系统调用之后加 1
			t := int64(_p_.syscalltick)
			if int64(pd.syscalltick) != t {
				// pd.syscalltick != _p_.syscalltick，说明已经不是上次观察到的系统调用了，
				// 而是另外一次系统调用，所以需要重新记录 tick 和 when 值
				pd.syscalltick = uint32(t)
				pd.syscallwhen = now
				continue
			}
	
			// 只要满足下面三个条件中的任意一个，则抢占该 p，否则不抢占
			// 1. p 的运行队列里面有等待运行的 goroutine
			// 2. 没有无所事事的 p
			// 3. 从上一次监控线程观察到 p 对应的 m 处于系统调用之中到现在已经超过 10 毫秒
			if runqempty(_p_) && atomic.Load(&sched.nmspinning)+atomic.Load(&sched.npidle) > 0 && pd.syscallwhen+10*1000*1000 > now {
				continue
			}
			
			incidlelocked(-1)
			if atomic.Cas(&_p_.status, s, _Pidle) {
				// ……………………
				n++
				_p_.syscalltick++
				// 寻找一新的 m 接管 p
				handoffp(_p_)
			}
			incidlelocked(1)
		} else if s == _Prunning {
			// P 处于运行状态，检查是否运行得太久了
			// Preempt G if it's running for too long.
			// 每发生一次调度，调度器 ++ 该值
			t := int64(_p_.schedtick)
			if int64(pd.schedtick) != t {
				pd.schedtick = uint32(t)
				pd.schedwhen = now
				continue
			}
			//pd.schedtick == t 说明(pd.schedwhen ～ now)这段时间未发生过调度
			// 这段时间是同一个goroutine一直在运行，检查是否连续运行超过了 10 毫秒
			if pd.schedwhen+forcePreemptNS > now {
				continue
			}
			// 连续运行超过 10 毫秒了，发起抢占请求
			preemptone(_p_)
		}
	}
	return uint32(n)
}
```

从代码来看，主要会对处于 `_Psyscall` 和 `_Prunning` 状态的 p 进行抢占。

# 抢占进行系统调用的 P

当 P 处于 `_Psyscall` 状态时，表明对应的 goroutine 正在进行系统调用。如果抢占 p，需要满足几个条件：

1. p 的本地运行队列里面有等待运行的 goroutine。这时 p 绑定的 g 正在进行系统调用，无法去执行其他的 g，因此需要接管 p 来执行其他的 g。

2. 没有“无所事事”的 p。`sched.nmspinning` 和 `sched.npidle` 都为 0，这就意味着没有“找工作”的 m，也没有空闲的 p，大家都在“忙”，可能有很多工作要做。因此要抢占当前的 p，让它来承担一部分工作。

3. 从上一次监控线程观察到 p 对应的 m 处于系统调用之中到现在已经超过 10 毫秒。这说明系统调用所花费的时间较长，需要对其进行抢占，以此来使得 `retake` 函数返回值不为 0，这样，会保持 sysmon 线程 20 us 的检查周期，提高 sysmon 监控的实时性。

注意，原代码是用的三个与条件，三者都要满足才会执行下面的 continue，也就是不进行抢占。因此要想进行抢占的话，只需要三个条件有一个不满足就行了。于是就有了上述三种情况。

确定要抢占当前 p 后，先使用原子操作将 p 的状态修改为 `_Pidle`，最后调用 `handoffp` 进行抢占。

```golang
func handoffp(_p_ *p) {
	// 如果 p 本地有工作或者全局有工作，需要绑定一个 m
	if !runqempty(_p_) || sched.runqsize != 0 {
		startm(_p_, false)
		return
	}

	// ……………………

	// 所有其它 p 都在运行 goroutine，说明系统比较忙，需要启动 m
	if atomic.Load(&sched.nmspinning)+atomic.Load(&sched.npidle) == 0 && atomic.Cas(&sched.nmspinning, 0, 1) { // TODO: fast atomic
		// p 没有本地工作，启动一个自旋 m 来找工作
		startm(_p_, true)
		return
	}
	lock(&sched.lock)

	// ……………………

	// 全局队列有工作
	if sched.runqsize != 0 {
		unlock(&sched.lock)
		startm(_p_, false)
		return
	}

	// ……………………

	// 没有工作要处理，把 p 放入全局空闲队列
	pidleput(_p_)
	unlock(&sched.lock)
}
```

`handoffp` 再次进行场景判断，以调用 `startm` 启动一个工作线程来绑定 p，使得整体工作继续推进。

当 p 的本地运行队列或全局运行队列里面有待运行的 goroutine，说明还有很多工作要做，调用 `startm(_p_, false)` 启动一个 m 来结合 p，继续工作。

当除了当前的 p 外，其他所有的 p 都在运行 goroutine，说明天下太平，每个人都有自己的事做，唯独自己没有。为了全局更快地完成工作，需要启动一个 m，且要使得 m 处于自旋状态，和 p 结合之后，尽快找到工作。

最后，如果实在没有工作要处理，就将 p 放入全局空闲队列里。

我们接着来看 `startm` 函数都做了些什么：

```golang
// runtime/proc.go
// 
// 调用 m 来绑定 p，如果没有 m，那就新建一个
// 如果 p 为空，那就尝试获取一个处于空闲状态的 p，如果找到 p，那就什么都不做
func startm(_p_ *p, spinning bool) {
	lock(&sched.lock)
	if _p_ == nil {
		// 没有指定 p 则需要从全局空闲队列中获取一个 p
		_p_ = pidleget()
		if _p_ == nil {
			unlock(&sched.lock)
			if spinning {
				// 如果找到 p，放弃。还原全局处于自旋状态的 m 的数量
				if int32(atomic.Xadd(&sched.nmspinning, -1)) < 0 {
					throw("startm: negative nmspinning")
				}
			}
			// 没有空闲的 p，直接返回
			return
		}
	}

	// 从 m 空闲队列中获取正处于睡眠之中的工作线程，
	// 所有处于睡眠状态的 m 都在此队列中
	mp := mget()
	unlock(&sched.lock)
	if mp == nil {
		// 如果没有找到 m
		var fn func()
		if spinning {
			// The caller incremented nmspinning, so set m.spinning in the new M.
			fn = mspinning
		}
		// 创建新的工作线程
		newm(fn, _p_)
		return
	}
	if mp.spinning {
		throw("startm: m is spinning")
	}
	if mp.nextp != 0 {
		throw("startm: m has p")
	}
	if spinning && !runqempty(_p_) {
		throw("startm: p has runnable gs")
	}
	// The caller incremented nmspinning, so set m.spinning in the new M.
	mp.spinning = spinning
	// 设置 m 马上要结合的 p
	mp.nextp.set(_p_)
	// 唤醒 m
	notewakeup(&mp.park)
}
```

首先处理 p 为空的情况，直接从全局空闲 p 队列里找，如果没找到，则直接返回。如果设置了 spinning 为 true 的话，还需要还原全局的处于自旋状态的 m 的数值：`&sched.nmspinning` 。

搞定了 p，接下来看 m。先调用 `mget` 函数从全局空闲的 m 队列里获取一个 m，如果没找到 m，则要调用 newm 新创建一个 m，并且如果设置了 spinning 为 true 的话，先要设置好 mstartfn：

```golang
func mspinning() {
	// startm's caller incremented nmspinning. Set the new M's spinning.
	getg().m.spinning = true
}
```

这样，启动 m 后，在 mstart1 函数里，进入 schedule 循环前，执行 mstartfn 函数，使得 m 处于自旋状态。

接下来是正常情况下（找到了 p 和 m）的处理：

```golang
mp.spinning = spinning
// 设置 m 马上要结合的 p
mp.nextp.set(_p_)
// 唤醒 m
notewakeup(&mp.park)
```

设置 nextp 为找到的 p，调用 `notewakeup` 唤醒 m。之前我们讲 findrunnable 函数的时候，对于最后没有找到工作的 m，我们调用 `notesleep(&_g_.m.park)`，使得 m 进入睡眠状态。现在终于有工作了，需要老将出山，将其唤醒：

```golang
// src/runtime/lock_futex.go
func notewakeup(n *note) {
	// 设置 n.key = 1, 被唤醒的线程通过查看该值是否等于 1 
	// 来确定是被其它线程唤醒还是意外从睡眠中苏醒
	old := atomic.Xchg(key32(&n.key), 1)
	if old != 0 {
		print("notewakeup - double wakeup (", old, ")\n")
		throw("notewakeup - double wakeup")
	}
	futexwakeup(key32(&n.key), 1)
}
```

> `notewakeup` 函数首先使用 `atomic.Xchg` 设置 `note.key` 值为 1，这是为了使被唤醒的线程可以通过查看该值是否等于 1 来确定是被其它线程唤醒还是意外从睡眠中苏醒了过来。

> 如果该值为 1 则表示是被唤醒的，可以继续工作，但如果该值为 0 则表示是意外苏醒，需要再次进入睡眠。

调用 `futexwakeup` 来唤醒工作线程，它和 `futexsleep` 是相对的。

```golang
func futexwakeup(addr *uint32, cnt uint32) {
	// 调用 futex 函数唤醒工作线程
	ret := futex(unsafe.Pointer(addr), _FUTEX_WAKE, cnt, nil, nil, 0)
	if ret >= 0 {
		return
	}
	
    // ……………………
    
}
```

`futex` 由汇编语言实现，前面已经分析过，这里就不重复了。主要内容就是先准备好参数，然后进行系统调用，由内核唤醒线程。

> 内核在完成唤醒工作之后当前工作线程从内核返回到 futex 函数继续执行 SYSCALL 指令之后的代码并按函数调用链原路返回，继续执行其它代码。

> 而被唤醒的工作线程则由内核负责在适当的时候调度到 CPU 上运行。

# 抢占长时间运行的 P
我们知道，Go scheduler 采用的是一种称为协作式的抢占式调度，就是说并不强制调度，大家保持协作关系，互相信任。对于长时间运行的 P，或者说绑定在 P 上的长时间运行的 goroutine，sysmon 会检测到这种情况，然后设置一些标志，表示 goroutine 自己让出 CPU 的执行权，给其他 goroutine 一些机会。

接下来我们就来分析当 P 处于 `_Prunning` 状态的情况。`sysmon` 扫描每个 p 时，都会记录下当前调度器调度的次数和当前时间，数据记录在结构体：

```golang
type sysmontick struct {
	schedtick   uint32
	schedwhen   int64
	syscalltick uint32
	syscallwhen int64
}
```

前面两个字段记录调度器调度的次数和时间，后面两个字段记录系统调用的次数和时间。

在下一次扫描时，对比 sysmon 记录下的 p 的调度次数和时间，与当前 p 自己记录下的调度次数和时间对比，如果一致。说明 P 在这一段时间内一直在运行同一个 goroutine。那就来计算一下运行时间是否太长了。

如果发现运行时间超过了 10 ms，则要调用 `preemptone(_p_)` 发起抢占的请求：

```golang
func preemptone(_p_ *p) bool {
	mp := _p_.m.ptr()
	if mp == nil || mp == getg().m {
		return false
	}
	// 被抢占的 goroutine
	gp := mp.curg
	if gp == nil || gp == mp.g0 {
		return false
	}

	// 设置抢占标志
	gp.preempt = true

	// 在 goroutine 内部的每次调用都会比较栈顶指针和 g.stackguard0，
	// 来判断是否发生了栈溢出。stackPreempt 非常大的一个数，比任何栈都大
	// stackPreempt = 0xfffffade
	gp.stackguard0 = stackPreempt
	return true
}
```

基本上只是将 stackguard0 设置了一个很大的值，而检查 stackguard0 的地方在函数调用前的一段汇编代码里进行。

举一个简单的例子：

```golang
package main

import "fmt"

func main() {
	fmt.Println("hello qcrao.com!")
}
```

执行命令：

```shell
go tool compile -S main.go
```

得到汇编代码：

```asm
"".main STEXT size=120 args=0x0 locals=0x48
	0x0000 00000 (test26.go:5)	TEXT	"".main(SB), $72-0
    0x0000 00000 (test26.go:5)	MOVQ	(TLS), CX
    0x0009 00009 (test26.go:5)	CMPQ	SP, 16(CX)
    0x000d 00013 (test26.go:5)	JLS	113
    0x000f 00015 (test26.go:5)	SUBQ	$72, SP
	0x0013 00019 (test26.go:5)	MOVQ	BP, 64(SP)
	0x0018 00024 (test26.go:5)	LEAQ	64(SP), BP
    0x001d 00029 (test26.go:5)	FUNCDATA	$0, gclocals·69c1753bd5f81501d95132d08af04464(SB)
    0x001d 00029 (test26.go:5)	FUNCDATA	$1, gclocals·e226d4ae4a7cad8835311c6a4683c14f(SB)
	0x001d 00029 (test26.go:6)	MOVQ	$0, ""..autotmp_0+48(SP)
    0x0026 00038 (test26.go:6)	MOVQ	$0, ""..autotmp_0+56(SP)
	0x002f 00047 (test26.go:6)	LEAQ	type.string(SB), AX
	0x0036 00054 (test26.go:6)	MOVQ	AX, ""..autotmp_0+48(SP)
	0x003b 00059 (test26.go:6)	LEAQ	"".statictmp_0(SB), AX
	0x0042 00066 (test26.go:6)	MOVQ	AX, ""..autotmp_0+56(SP)
	0x0047 00071 (test26.go:6)	LEAQ	""..autotmp_0+48(SP), AX
	0x004c 00076 (test26.go:6)	MOVQ	AX, (SP)
	0x0050 00080 (test26.go:6)	MOVQ	$1, 8(SP)
    0x0059 00089 (test26.go:6)	MOVQ	$1, 16(SP)
	0x0062 00098 (test26.go:6)	PCDATA	$0, $1
	0x0062 00098 (test26.go:6)	CALL	fmt.Println(SB)
	0x0067 00103 (test26.go:7)	MOVQ	64(SP), BP
	0x006c 00108 (test26.go:7)	ADDQ	$72, SP
    0x0070 00112 (test26.go:7)	RET
    0x0071 00113 (test26.go:7)	NOP
    0x0071 00113 (test26.go:5)	PCDATA	$0, $-1
	0x0071 00113 (test26.go:5)	CALL	runtime.morestack_noctxt(SB)
	0x0076 00118 (test26.go:5)	JMP	0
```

以前看这段代码的时候会直接跳过前面的几行代码，看不懂。这次能看懂了！所以，那些暂时看不懂的，先放一放，没关系，让子弹飞一会儿，很多东西回过头再来看就会豁然开朗，这就是一个很好的例子。

```asm
0x0000 00000 (test26.go:5)	MOVQ	(TLS), CX
```

将本地存储 tls 保存到 CX 寄存器中，（TLS）表示它所关联的 g，这里就是前面所讲到的 main gouroutine。

```asm
0x0009 00009 (test26.go:5)	CMPQ	SP, 16(CX)
```

比较 SP 寄存器（代表当前 main goroutine 的栈顶寄存器）和 16(CX)，我们看下 g 结构体：

```golang
type g struct {
	// goroutine 使用的栈
	stack       stack   // offset known to runtime/cgo
	// 用于栈的扩张和收缩检查
	stackguard0 uintptr // offset known to liblink
	// ……………………
}
```

对象 g 的第一个字段是 stack 结构体：

```golang
type stack struct {
	lo uintptr
	hi uintptr
}
```

共 16 字节。而 `16(CX)` 表示 g 对象的第 16 个字节，跳过了 g 的第一个字段，也就是 `g.stackguard0` 字段。

如果 SP 小于 g.stackguard0，这是必然的，因为前面已经把 g.stackguard0 设置成了一个非常大的值，因此跳转到了 113 行。

```asm
0x0071 00113 (test26.go:7)	NOP
0x0071 00113 (test26.go:5)	PCDATA	$0, $-1
0x0071 00113 (test26.go:5)	CALL	runtime.morestack_noctxt(SB)
0x0076 00118 (test26.go:5)	JMP	0
```

调用 `runtime.morestack_noctxt` 函数：

```asm
// src/runtime/asm_amd64.s
TEXT runtime·morestack_noctxt(SB),NOSPLIT,$0
	MOVL	$0, DX
	JMP	runtime·morestack(SB)
```

直接跳转到 `morestack` 函数：

```asm
TEXT runtime·morestack(SB),NOSPLIT,$0-0
    // Cannot grow scheduler stack (m->g0).
    get_tls(CX)
    // BX = g，g 表示 main goroutine
    MOVQ	g(CX), BX
    // BX = g.m
    MOVQ	g_m(BX), BX
    // SI = g.m.g0
    MOVQ	m_g0(BX), SI
    CMPQ	g(CX), SI
    JNE	3(PC)
    CALL	runtime·badmorestackg0(SB)
    INT	$3

	// ……………………

	// Set g->sched to context in f.
	// 将函数的返回地址保存到 AX 寄存器
	MOVQ	0(SP), AX // f's PC
	// 将函数的返回地址保存到 g.sched.pc
	MOVQ	AX, (g_sched+gobuf_pc)(SI)
	// g.sched.g = g
	MOVQ	SI, (g_sched+gobuf_g)(SI)
	// 取地址操作符，调用 morestack_noctxt 之前的 rsp
	LEAQ	8(SP), AX // f's SP
	// 将 main 函数的栈顶地址保存到 g.sched.sp
	MOVQ	AX, (g_sched+gobuf_sp)(SI)
	// 将 BP 寄存器保存到 g.sched.bp
	MOVQ	BP, (g_sched+gobuf_bp)(SI)
	// newstack will fill gobuf.ctxt.

	// Call newstack on m->g0's stack.
	// BX = g.m.g0
	MOVQ	m_g0(BX), BX
	// 将 g0 保存到本地存储 tls
	MOVQ	BX, g(CX)
	// 把 g0 栈的栈顶寄存器的值恢复到 CPU 的寄存器 SP，达到切换栈的目的，下面这一条指令执行之前，
    // CPU 还是使用的调用此函数的 g 的栈，执行之后 CPU 就开始使用 g0 的栈了
	MOVQ	(g_sched+gobuf_sp)(BX), SP
	// 准备参数
	PUSHQ	DX	// ctxt argument
	// 不返回
	CALL	runtime·newstack(SB)
	MOVQ	$0, 0x1003	// crash if newstack returns
	POPQ	DX	// keep balance check happy
	RET
```

主要做的工作就是将当前 goroutine，也就是 main goroutine 的和调度相关的信息保存到 g.sched 中，以便在调度到它执行时，可以恢复。

最后，将 g0 的地址保存到 tls 本地存储，并且切到 g0 栈执行之后的代码。继续调用 newstack 函数：

```golang
func newstack(ctxt unsafe.Pointer) {
	// thisg = g0
	thisg := getg()
	
	// ……………………

	// gp = main goroutine
	gp := thisg.m.curg
	// Write ctxt to gp.sched. We do this here instead of in
	// morestack so it has the necessary write barrier.
	gp.sched.ctxt = ctxt

	// ……………………

	morebuf := thisg.m.morebuf
	thisg.m.morebuf.pc = 0
	thisg.m.morebuf.lr = 0
	thisg.m.morebuf.sp = 0
	thisg.m.morebuf.g = 0

	// 检查 g.stackguard0 是否被设置成抢占标志
	preempt := atomic.Loaduintptr(&gp.stackguard0) == stackPreempt

	if preempt {
		if thisg.m.locks != 0 || thisg.m.mallocing != 0 || thisg.m.preemptoff != "" || thisg.m.p.ptr().status != _Prunning {
			// 还原 stackguard0 为正常值，表示我们已经处理过抢占请求了
			gp.stackguard0 = gp.stack.lo + _StackGuard
			// 不抢占，调用 gogo 继续运行当前这个 g，不需要调用 schedule 函数去挑选另一个 goroutine
			gogo(&gp.sched) // never return
		}
	}

	// ……………………

	if preempt {
		if gp == thisg.m.g0 {
			throw("runtime: preempt g0")
		}
		if thisg.m.p == 0 && thisg.m.locks == 0 {
			throw("runtime: g is running but p is not")
		}
		// Synchronize with scang.
		casgstatus(gp, _Grunning, _Gwaiting)

		// ……………………

		// Act like goroutine called runtime.Gosched.
		// 修改为 running，调度起来运行
		casgstatus(gp, _Gwaiting, _Grunning)
		// 调用 gopreempt_m 把 gp 切换出去
		gopreempt_m(gp) // never return
	}

	// ……………………
}
```

去掉了很多暂时还看不懂的地方，留到后面再研究。只关注有关抢占相关的。第一次判断 preempt 标志是 true 时，检查了 g 的状态，发现不能抢占，例如它所绑定的 P 的状态不是 `_Prunning`，那就恢复它的 stackguard0 字段，下次就不会走这一套流程了。然后，调用 `gogo(&gp.sched)` 继续执行当前的 goroutine。

中间又处理了很多判断流程，再次判断 preempt 标志是 true 时，调用 `gopreempt_m(gp)` 将 gp 切换出去。

```golang
func gopreempt_m(gp *g) {
	if trace.enabled {
		traceGoPreempt()
	}
	goschedImpl(gp)
}
```

最终调用 `goschedImpl` 函数：

```golang
func goschedImpl(gp *g) {
	status := readgstatus(gp)
	if status&^_Gscan != _Grunning {
		dumpgstatus(gp)
		throw("bad g status")
	}
	// 更改 gp 的状态
	casgstatus(gp, _Grunning, _Grunnable)
	// 解除 m 和 g 的关系
	dropg()
	lock(&sched.lock)
	// 将 gp 放入全局可运行队列
	globrunqput(gp)
	unlock(&sched.lock)

	// 进入新一轮的调度循环
	schedule()
}
```

将 gp 的状态改为 `_Grunnable`，放入全局可运行队列，等待下次有 m 来全局队列找工作时才能继续运行，毕竟你已经运行这么长时间了，给别人一点机会嘛。

最后，调用 `schedule()` 函数进入新一轮的调度循环，会找出一个 goroutine 来运行，永不返回。

这样，关于 sysmon 线程在关于调度这块到底做了啥，我们已经回答完了。总结一下：

1. 抢占处于系统调用的 P，让其他 m 接管它，以运行其他的 goroutine。
2. 将运行时间过长的 goroutine 调度出去，给其他 goroutine 运行的机会。