# GC 的认识

## 1. 什么是 GC，有什么作用？

`GC`，全称 `Garbage Collection`，即垃圾回收，是一种自动内存管理的机制。

当程序向操作系统申请的内存不再需要时，垃圾回收主动将其回收并供其他代码进行内存申请时候复用，或者将其归还给操作系统，这种针对内存级别的资源的自动回收过程，即为垃圾回收，而负责垃圾回收的程序组件，即为垃圾回收器。

垃圾回收其实一个完美的 “Simplicity is Complicated” 的例子。一方面，程序员受益于 GC，无需操心、也不再需要对内存进行手动的申请和释放操作，GC 在程序运行时自动释放残留过多的内存。另一方面，GC 对程序员几乎不可见，仅在程序需要进行特殊优化，通过提供可调控的 API，对 GC 的运行时机、运行开销进行把控的时候才得以现身。

通常，垃圾回收器的执行过程被划分为两个半独立的组件：

- 赋值器（Mutator）：这一名称本质上是在指代用户态的代码。因为对垃圾回收器而言，用户态的代码仅仅只是在修改对象之间的引用关系，也就是在对象图（对象之间引用关系的一个有向图）上进行操作。
- 回收器（Collector）：负责执行垃圾回收的代码。

## 2. 根对象到底是什么？

根对象在垃圾回收的术语中又叫做根集合，它是垃圾回收器在标记过程时最先检查的对象，包括：

1. 全局变量：程序在编译期就能确定的那些存在于程序整个生命周期的变量
2. 执行栈：每个 goroutine 都包含自己的执行栈，这些执行栈上包含栈上的变量及指向分配的内存区块的指针
3. 寄存器：寄存器的值可能表示一个指针，参与计算的这些指针可能指向某些赋值器分配的内存区块

## 3. 常见的 GC 实现方式有哪些？Go 语言的 GC 使用的什么？

所有的 GC 算法其存在形式可以归结为追踪（Tracing）和引用计数（Reference Counting）这两种形式的混合运用。

- 追踪式 GC 
  + 核心思想：从根对象出发，根据对象之间的引用信息，一步步推进直到扫描完毕整个堆、并确定需要保留的对象，从而回收所有可以回收的对象。例如 Go、 Java、V8 对 JavaScript 的实现等均为追踪式 GC。
- 引用计数式 GC
  + 核心思相想：每个对象自身包含一个被引用的计数器，当计数器归零时自动得到回收。因为此方法缺陷较多，在追求高性能时通常不被应用，例如：Python、Objective-C 等均为引用计数式 GC。

目前比较常见的 GC 实现方式包括：

- 追踪式，分为多种不同类型，例如：
  - 标记清扫：从根对象出发，将确定存活的对象进行标记，并清扫可以回收的对象。
  - 标记整理：为了解决内存碎片问题而提出，在标记过程中，将对象尽可能整理到一块连续的内存上。
  - 增量式：将标记与清扫的过程分批执行，每次执行很小的部分，从而增量的推进垃圾回收，达到近似实时、几乎无停顿的目的。
  - 增量整理：在增量式的基础上，增加对对象的整理过程。
  - 分代式：将对象根据存活时间的长短进行分类，存活时间小于某个值的为年轻代，存活时间大于某个值的为老年代，永远不会参与回收的对象为永久代。并根据分代假设（如果一个对象存活时间不长则倾向于被回收，如果一个对象已经存活很长时间则倾向于存活更长时间）对对象进行回收。
- 引用计数：根据对象自身的引用的计数来回收，当引用数归零时立即回收。

关于各类方法的详细介绍及其实现不在本文中详细讨论。对于 Go 而言，Go 的 GC 目前使用的是无分代（对象没有代际之分）、不整理（回收过程中不对对象进行移动与整理）、并发（与用户代码并发执行）的三色标记清扫算法。[原因](https://groups.google.com/d/msg/golang-nuts/KJiyv2mV2pU/wdBUH1mHCAAJ)在于：

1. 对象整理的优势是解决顺序分配器产生的内存碎片问题，但 Go 运行时的分配算法基于 tcmalloc，在多线程环境下，对对象进行整理不会带来实质性的性能提升。
2. 分代 GC 依赖分代假设，即 GC 将主要的回收目标放在新创建的对象上，而非频繁检查所有对象。但 Go 的编译器会将大部分新生对象存储在栈上（栈直接被回收），并利用逃逸分析直接将对象分配到堆中，因为需要逃逸的对象通常是那些需要长期存在的对象，当 goroutine 死亡后栈也会被直接回收，进而分代假设并没有带来直接优势。除此之外，由于 Go 的垃圾回收器与用户代码并发执行，从而 STW 与对象的代际没有关系，并着重优化如何更好的让 GC 与用户代码并发执行（即使用适当的 CPU 来执行垃圾回收），而非关注在减少停顿时间这一单一目标上。

## 4. 三色标记法是什么？

理解**三色标记法**的关键是理解对象的**三色抽象**以及**波面（wavefront）推进**这两个概念。三色抽象只是一种描述追踪式回收器的方法，在实践中并没有实际含义，它的重要作用在于从逻辑上严密推导标记清理这种垃圾回收方法的正确性，也就是说，当我们谈及三色标记法时，通常指标记清扫的垃圾回收。

从垃圾回收器的视角来看，三色抽象规定了三种不同类型的对象，并用不同的颜色相称：

- 白色对象（可能死亡）：未被回收器访问到的对象。在回收开始阶段，所有对象均为白色，当回收结束后，白色对象均不可达。
- 灰色对象（波面）：已被回收器访问到的对象，但回收器需要对其中的一个或多个指针进行扫描，因为他们可能还指向白色对象。
- 黑色对象（确定存活）：已被回收器访问到的对象，其中所有字段都已扫描，黑色对象中任何一个指针都不可能直接指向白色对象。

这样三种不变性所定义的回收过程其实是一个**波面**不断前进的过程，这个波面同时也是黑色对象和白色对象的边界，灰色对象就是这个波面。

当垃圾回收开始时，只有白色对象。当标记过程开始进行时，灰色对象开始出现（着色），这时候波面便开始扩大。当一个对象的所有子节点均完成扫描时，会被着色为黑色。当整个堆遍历完成时，只剩下黑色和白色对象，这时的黑色对象为可达对象，即存活；而白色对象为不可达对象，即死亡。这个过程可以视为以灰色对象为波面，将黑色对象和白色对象分离，使波面不断向前推进，直到所有可达的灰色对象都变为黑色对象为止的过程。如下图所示：

![三色标记法全貌](https://user-images.githubusercontent.com/7698088/70995777-505fc800-210c-11ea-9f6d-3549f884ff46.png)

图中展示了根对象、可达对象、不可达对象，黑、灰、白对象以及波面之间的关系。

## 5. STW 是什么，Go 为什么要一直尝试减少 STW 时间，真那么有用吗？

`STW` 是 `Stop the World` 的缩写，即万物静止，是指在垃圾回收过程中为了保证实现的正确性、防止无止境的内存增长等问题而不可避免的需要停止赋值器进一步操作对象图的一段过程。

在这个过程中整个用户代码被停止或者放缓执行，可想 `STW` 越长，对用户代码造成的影响（例如延迟）就越大，早期 Go 对垃圾回收器的实现中 `STW` 长达几百毫秒，对时间敏感的实时通信等应用程序会造成巨大的影响。我们来看一个例子：

```go
package main

import (
	"runtime"
	"time"
)

func main() {
	go func() {
		for {
		}
	}()

	time.Sleep(time.Millisecond)
	runtime.GC()
	println("OK")
}
```

上面的这个程序在 Go 1.14 以前永远都不会输出 `OK`，其罪魁祸首是 STW 无限制的被延长。

尽管 STW 如今已经优化到了半毫秒级别以下，但这个程序被卡死原因在于仍然是 STW 导致的。原因在于，GC 在进入 STW 时，需要等待让所有的用户态代码停止，但是 `for {}` 所在的 goroutine 永远都不会被中断，从而停留在 STW 阶段。实际实践中也是如此，当程序的某个 goroutine 长时间得不到停止，强行拖慢 STW，这种情况下造成的影响（卡死）是非常可怕的。好在自 Go 1.14 之后，这类 goroutine 能够被异步的抢占，从而使得 STW 的时间如同普通程序那样，不会超过半个毫秒，程序也不会因为仅仅等待一个 goroutine 的停止而停顿在 STW 阶段。

## 6. 如何观察 Go GC？

我们以下面的程序为例，先使用四种不同的方式来介绍如何观察 GC，
并在后面的问题中通过几个详细的例子再来讨论如何优化 GC。

```go
package main

func allocate() {
	_ = make([]byte, 1<<20)
}

func main() {
	for n := 1; n < 100000; n++ {
		allocate()
	}
}
```

### 方式1：`GODEBUG=gctrace=1`

我们首先可以通过

```
$ go1.14 build -o main
$ GODEBUG=gctrace=1 ./main

gc 1 @0.000s 2%: 0.009+0.23+0.004 ms clock, 0.11+0.083/0.019/0.14+0.049 ms cpu, 4->6->2 MB, 5 MB goal, 12 P
scvg: 8 KB released
scvg: inuse: 3, idle: 60, sys: 63, released: 57, consumed: 6 (MB)
gc 2 @0.001s 2%: 0.018+1.1+0.029 ms clock, 0.22+0.047/0.074/0.048+0.34 ms cpu, 4->7->3 MB, 5 MB goal, 12 P
scvg: inuse: 3, idle: 60, sys: 63, released: 56, consumed: 7 (MB)
gc 3 @0.003s 2%: 0.018+0.59+0.011 ms clock, 0.22+0.073/0.008/0.042+0.13 ms cpu, 5->6->1 MB, 6 MB goal, 12 P
scvg: 8 KB released
scvg: inuse: 2, idle: 61, sys: 63, released: 56, consumed: 7 (MB)
gc 4 @0.003s 4%: 0.019+0.70+0.054 ms clock, 0.23+0.051/0.047/0.085+0.65 ms cpu, 4->6->2 MB, 5 MB goal, 12 P
scvg: 8 KB released
scvg: inuse: 3, idle: 60, sys: 63, released: 56, consumed: 7 (MB)
scvg: 8 KB released
scvg: inuse: 4, idle: 59, sys: 63, released: 56, consumed: 7 (MB)
gc 5 @0.004s 12%: 0.021+0.26+0.49 ms clock, 0.26+0.046/0.037/0.11+5.8 ms cpu, 4->7->3 MB, 5 MB goal, 12 P
scvg: inuse: 5, idle: 58, sys: 63, released: 56, consumed: 7 (MB)
gc 6 @0.005s 12%: 0.020+0.17+0.004 ms clock, 0.25+0.080/0.070/0.053+0.051 ms cpu, 5->6->1 MB, 6 MB goal, 12 P
scvg: 8 KB released
scvg: inuse: 1, idle: 62, sys: 63, released: 56, consumed: 7 (MB)
```

在这个日志中可以观察到两类不同的信息：

```
gc 1 @0.000s 2%: 0.009+0.23+0.004 ms clock, 0.11+0.083/0.019/0.14+0.049 ms cpu, 4->6->2 MB, 5 MB goal, 12 P
gc 2 @0.001s 2%: 0.018+1.1+0.029 ms clock, 0.22+0.047/0.074/0.048+0.34 ms cpu, 4->7->3 MB, 5 MB goal, 12 P
...
```

以及

```
scvg: 8 KB released
scvg: inuse: 3, idle: 60, sys: 63, released: 57, consumed: 6 (MB)
scvg: inuse: 3, idle: 60, sys: 63, released: 56, consumed: 7 (MB)
...
```

对于用户代码向运行时申请内存产生的垃圾回收：

```
gc 2 @0.001s 2%: 0.018+1.1+0.029 ms clock, 0.22+0.047/0.074/0.048+0.34 ms cpu, 4->7->3 MB, 5 MB goal, 12 P
```

含义由下表所示：

|字段|含义|
|:--|:--|
|gc 2| 第二个 GC 周期 |
|0.001| 程序开始后的 0.001 秒 |
|2%| 该 GC 周期中 CPU 的使用率 |
|0.018| 标记开始时， STW 所花费的时间（wall clock）|
|1.1| 标记过程中，并发标记所花费的时间（wall clock）|
|0.029| 标记终止时， STW 所花费的时间（wall clock）|
|0.22| 标记开始时 STW 所花费的时间（cpu time）|
|0.047| 标记过程中，标记辅助所花费的时间（cpu time）|
|0.074| 标记过程中，并发标记所花费的时间（cpu time）|
|0.048| 标记过程中，GC 空闲的时间（cpu time）|
|0.34| 标记终止时， STW 所花费的时间（cpu time）|
|4| 标记开始时，堆的大小的实际值 |
|7| 标记结束时，堆的大小的实际值 |
|3| 标记结束时，标记为存活的对象大小 |
|5| 标记结束时，堆的大小的预测值 |
|12| P 的数量 |

> wall clock 是指开始执行到完成所经历的实际时间，包括其他程序和本程序所消耗的时间；
> cpu clock 是指特定程序使用 CPU 的时间；
> 他们存在以下关系：
> 
> - wall clock < cpu time: 充分利用多核
> - wall clock ≈ cpu time: 未并行执行
> - wall clock > cpu time: 多核优势不明显

对于运行时向操作系统申请内存产生的垃圾回收（向操作系统归还多余的内存）：

```
scvg: 8 KB released
scvg: inuse: 3, idle: 60, sys: 63, released: 57, consumed: 6 (MB)
```

含义由下表所示：

|字段|含义|
|:--|:--|
|8 KB released| 向操作系统归还了 8 KB 内存 |
|3| 已经分配给用户代码、正在使用的总内存大小 (MB) |
|60| 空闲以及等待归还给操作系统的总内存大小（MB）|
|63| 通知操作系统中保留的内存大小（MB）|
|57| 已经归还给操作系统的（或者说还未正式申请）的内存大小（MB）|
|6| 已经从操作系统中申请的内存大小（MB）|

### 方式2：`go tool trace`

`go tool trace` 的主要功能是将统计而来的信息以一种可视化的方式
展示给用户。要使用此方式，可以通过调用 trace API：

```go
package main

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()
	(...)
}
```

并通过 

```
$ go tool trace trace.out
2019/12/30 15:50:33 Parsing trace...
2019/12/30 15:50:38 Splitting trace...
2019/12/30 15:50:45 Opening browser. Trace viewer is listening on http://127.0.0.1:51839
```

命令来启动可视化界面：

![](./assets/gc-trace.png)

选择第一个链接可以获得如下图示：

![](assets/gc-trace2.png)

右上角的问号可以打开帮助菜单，主要使用方式包括：

- w/s 键可以用于放大或者缩小视图
- a/d 键可以用户左右移动

### 方式3：`debug.ReadGCStats`

此方式可以通过代码的方式来直接实现对感兴趣指标的监控，例如我们希望每隔一秒钟监控一次
GC 的状态：

```go
func printGCStats() {
	t := time.NewTicker(time.Second)
	s := debug.GCStats{}
	for {
		select {
		case <-t.C:
			debug.ReadGCStats(&s)
			fmt.Printf("gc %d last@%v, PauseTotal %v\n", s.NumGC, s.LastGC, s.PauseTotal)
		}
	}
}
func main() {
	go printGCStats()
	(...)
}
```

我们能够看到如下输出：

```go
$ go run main.go

gc 4954 last@2019-12-30 15:19:37.505575 +0100 CET, PauseTotal 29.901171ms
gc 9195 last@2019-12-30 15:19:38.50565 +0100 CET, PauseTotal 77.579622ms
gc 13502 last@2019-12-30 15:19:39.505714 +0100 CET, PauseTotal 128.022307ms
gc 17555 last@2019-12-30 15:19:40.505579 +0100 CET, PauseTotal 182.816528ms
gc 21838 last@2019-12-30 15:19:41.505595 +0100 CET, PauseTotal 246.618502ms
```

### 方式4：`runtime.ReadMemStats`

除了使用 debug 包提供的方法外，还可以直接通过运行时的内存相关的 API 进行监控：

```go
func printMemStats() {
	t := time.NewTicker(time.Second)
	s := runtime.MemStats{}

	for {
		select {
		case <-t.C:
			runtime.ReadMemStats(&s)
			fmt.Printf("gc %d last@%v, next_heap_size@%vMB\n", s.NumGC, time.Unix(int64(time.Duration(s.LastGC).Seconds()), 0), s.NextGC/(1<<20))
		}
	}
}
func main() {
	go printMemStats()
	(...)
}
```

```go
$ go run main.go

gc 4887 last@2019-12-30 15:44:56 +0100 CET, next_heap_size@4MB
gc 10049 last@2019-12-30 15:44:57 +0100 CET, next_heap_size@4MB
gc 15231 last@2019-12-30 15:44:58 +0100 CET, next_heap_size@4MB
gc 20378 last@2019-12-30 15:44:59 +0100 CET, next_heap_size@6MB
```

当然，后两种方式能够监控的指标很多，读者可以自行查看 [`debug.GCStats`](https://golang.org/pkg/runtime/debug/#GCStats) 和 
[`runtime.MemStats`](https://golang.org/pkg/runtime/#MemStats) 的字段，这里不再赘述。

## 7. 有了 GC，为什么还会发生内存泄露？

在一个具有 GC 的语言中，常说的内存泄漏，用严谨的话来说应该是：预期的能很快被释放的内存由于附着在了长期存活的内存上、或生命期意外地被延长，从而预计能够立即回收的内存而长时间得不到回收。

在 Go 中，由于 goroutine 的存在，所谓的内存泄漏除了附着在长期对象上之外，还存在多种不同的形式。

### 形式1：预期能被快速释放的内存因被根对象引用而没有得到迅速释放

当有一个全局对象时，可能不经意间将某个变量附着在其上，且忽略的将其进行释放，则该内存永远不会得到释放。例如：

```go
var cache = map[interface{}]interface{}{}

func keepalloc() {
	for i := 0; i < 10000; i++ {
		m := make([]byte, 1<<10)
		cache[i] = m
	}
}
```

### 形式2：goroutine 泄漏

Goroutine 作为一种逻辑上理解的轻量级线程，需要维护用户代码的执行上下文信息。因此，它在运行过程中也需要消耗一定的内存来保存这类信息。而这些内存在目前版本（Go 1.14）的 Go 中是不会被释放的。因此，如果一个程序持续不断的产生新的 goroutine、且不结束执行已经创建的 goroutine 并复用这部分内存，就会造成内存泄漏的现象，例如：

```go
func keepalloc2() {
	for i := 0; i < 100000; i++ {
		go func() {
			select {}
		}()
	}
}
```

### 验证

我们可以通过如下形式来调用上述两个函数：

```go
package main

import (
	"os"
	"runtime/trace"
)

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()
	keepalloc()
	keepalloc2()
}
```

然后运行程序：

```shell
go run main.go
```

会看到程序中生成了 `trace.out` 文件，我们可以使用 `go tool trace trace.out`
命令观察到下图：

![](./assets/gc-leak1.png)

可以看到，途中的 Heap 在持续增长，没有内存被回收，产生了内存泄漏的现象。

值得一提的是，这种形式的 goroutine 泄漏还可能由 channel 泄漏导致。
而 channel 的泄漏本质上与 goroutine 泄漏存在直接联系。
channel 作为一种同步原语，会连接两个不同的 goroutine，
如果一个 goroutine 尝试向一个没有接收方的无缓冲 channel 发送消息，
则该 goroutine 会被永久的休眠，整个 goroutine 及其执行栈都得不到释放，例如：

```go
var ch = make(chan struct{})

func keepalloc3() {
	for i := 0; i < 100000; i++ {
		// 没有接收方，goroutine 会一直阻塞
		go func() { ch <- struct{}{} }()
	}
}
```

## 8. 并发标记清除法的难点是什么？

在没有用户态代码并发修改`三色抽象`的情况下，回收可以正常结束。
但是并发回收的根本问题在于，用户态代码在回收过程中会并发的更新对象图，
从而造成赋值器和回收器可能对对象图的结构产生不同的认知。
这时以一个固定的三色波面作为回收过程前进的边界则不再合理。

我们不妨考虑赋值器的写操作的例子：

| 时序 | 回收器               | 赋值器             | 说明                                                         |
| ---- | -------------------- | ------------------ | ------------------------------------------------------------ |
| 1    | shade(A, gray)       |                    | 回收器：根对象的子节点着色为灰色对象                         |
| 2    | shade(C, black) |                    | 回收器：当所有子节点着色为灰色后，将节点着为黑色             |
| 3    |                      | C.ref3 = C.ref2.ref1   | 赋值器：并发的修改了 C 的子节点                              |
| 4    |                      | A.ref1 = nil | 赋值器：并发的修改了 A 的子节点                              |
| 5    | shade(A.ref1, gray)  |                    | 回收器：进一步扫描着色为灰色对象的子节点着色为灰色对象，这时由于 `A.ref1` 为 `nil`，什么事情也没有发生 |
| 6    | shade(A, black)      |                    | 回收器：由于所有子节点均已标记，回收器也不会重新扫描已经被标记为黑色的对象，此时 A 被着色为黑色，`scan(A)` 什么也不会发生，进而 B 在此次回收过程中永远不会被标记为黑色，进而错误的回收。 |

- 初始状态：假设某个黑色对象 C 指向某个灰色对象 A ，而 A 指向白色对象 B；
- `C.ref3 = C.ref2.ref1`：而此时赋值器并发的将黑色对象 C 指向（ref3）了白色对象 B；
- `A.ref1 = nil`：并将灰色对象 A 对白色对象 B 的引用移除（ref2）；
- 最终状态：则在继续扫描的过程中，白色对象 B 永远不会被标记为黑色对象了（回收器不会重新扫描黑色对象），进而产生被错误回收的对象 B。

![gc-mutator](./assets/gc-mutator.png)

总而言之，并发标记清除中面临的一个根本问题就是如何保证标记与清除过程的正确性。

## 9. 什么是写屏障、混合写屏障，如何实现？

要讲清楚写屏障，就需要理解三色标记清扫算法中的**强弱不变性**以及**赋值器的颜色**，
理解他们需要一定的抽象思维。

写屏障是一个在并发垃圾回收器中才会出现的概念，垃圾回收器的正确性体现在：
_不应出现对象的丢失，也不应错误的回收还不需要回收的对象。_

可以证明，当以下两个条件同时满足时会破坏垃圾回收器的正确性：

- **条件 1**: 赋值器修改对象图，导致某一黑色对象引用白色对象；
- **条件 2**: 从灰色对象出发，到达白色对象的、未经访问过的路径被赋值器破坏。

因此只要能够避免其中任何一个条件，则不会出现对象丢失的情况，因为：

- 如果条件 1 被避免，则所有白色对象均被灰色对象引用，没有白色对象会被遗漏
- 如果条件 2 被避免，即便白色对象的指针写入到黑色对象中，但从灰色对象出发，总存在一条没有访问过的路径，从而找到到达白色对象的路径，白色对象最终不会被遗漏

我们不妨将三色不变性所定义的波面根据这两个条件进行削弱：

- 当满足原有的三色不变性定义（或上面的两个条件都不满足时）的情况称为**强三色不变性（strong tricolor invariant）**
<!-- 即不存在黑色对象指向白色对象的指针； -->
- 当赋值器令黑色对象引用白色对象时（满足条件 1 时）的情况称为**弱三色不变性（weak tricolor invariant）**
<!-- 即所有黑色对象引用的白色对象都处于灰色保护状态（直接或间接从灰色对象可达）。 -->

当赋值器进一步破坏灰色对象到达白色对象的路径时（进一步满足条件 2 时），即打破弱三色不变性，
则也就破坏了回收器的正确性；或者说，在破坏强弱三色不变性时必须引入额外的辅助操作。
弱三色不变形的好处在于：**只要存在未访问的能够到达白色对象的路径，就可以将黑色对象指向白色对象。**

如果我们考虑并发的用户态代码，回收器不允许同时停止所有赋值器，
就是涉及了存在的多个不同状态的赋值器。为了对概念加一明确，还需要换一个角度，把回收器视为对象，
赋值器视为影响回收器这一对象的实际行为（即影响 GC 周期的长短），从而还可以引入赋值器的颜色：

- 黑色赋值器：已经由回收器扫描过，不会再次对其进行扫描
- 灰色赋值器：尚未被回收器扫描过，或尽管已经扫描过但仍需要重新扫描

赋值器的颜色对回收周期的结束产生影响：

- 如果某种并发回收器允许灰色赋值器的存在，则必须在回收结束之前重新扫描对象图。
- 如果重新扫描过程中发现了新的灰色或白色对象，回收器还需要对新发现的对象进行追踪，但是在新追踪的过程中，赋值器仍然可能在其根中插入新的非黑色的引用，如此往复，直到重新扫描过程中没有发现新的白色或灰色对象。

从而在允许灰色赋值器存在的算法，最坏的情况下，
回收器只能将所有赋值器线程停止才能完成其跟对象的完整扫描，也就是我们所说的 STW。
为了确保强弱三色不变性的并发指针更新操作需要通过赋值器屏障技术来保证指针的读写操作一致。
因此我们所说的 Go 中的写屏障、混合写屏障，其实是指赋值器的写屏障，赋值器的写屏障用来
保证赋值器在进行指针写操作时，不会破坏弱三色不变性。

有两种非常经典的写屏障：Dijkstra 插入屏障和 Yuasa 删除屏障。

灰色赋值器的 Dijkstra 插入屏障的基本思想是避免满足条件 1：

```go
// 灰色赋值器 Dijkstra 插入屏障
func DijkstraWritePointer(slot *unsafe.Pointer, ptr unsafe.Pointer) {
    shade(ptr)
    *slot = ptr
}
```

为了防止黑色对象指向白色对象，应该假设 `*slot` 可能会变为黑色，为了确保 `ptr` 不会在将赋值为 `*slot` 前变为白色，
`shade(ptr)` 会先将指针 `ptr` 标记为灰色，进而避免了条件 1。
但是，由于并不清楚赋值器以后会不会将这个引用删除，因此还需要重新扫描来重新确定关系图，这时需要 STW，如图所示。

![](./assets/gc-wb-dijkstra.png)

Dijkstra 插入屏障的好处在于可以立刻开始并发标记，但由于产生了灰色赋值器，缺陷是需要标记终止阶段 STW 时进行重新扫描。

黑色赋值器的 Yuasa 删除屏障的基本思想是避免满足条件 2：

```go
// 黑色赋值器 Yuasa 屏障
func YuasaWritePointer(slot *unsafe.Pointer, ptr unsafe.Pointer) {
    shade(*slot)
    *slot = ptr
}
```

为了防止丢失从灰色对象到白色对象的路径，应该假设 `*slot` 可能会变为黑色，为了确保 `ptr` 不会在将赋值为 `*slot` 前变为白色，`shade(*slot)` 会先将 `*slot` 标记为灰色，进而该写操作总是创造了一条灰色到灰色或者灰色到白色对象的路径，进而避免了条件 2。

Yuasa 删除屏障的优势则在于不需要标记结束阶段的重新扫描，缺陷依然会产生丢失的对象，需要在标记开始前对整个对象图进行快照。

![](./assets/gc-wb-yuasa.png)

Go 在 1.8 的时候为了简化 GC 的流程，同时减少标记终止阶段的重扫成本，
将 Dijkstra 插入屏障和 Yuasa 删除屏障进行混合，形成混合写屏障。该屏障提出时的基本思想是：
**对正在被覆盖的对象进行着色，且如果当前栈未扫描完成，则同样对指针进行着色。**
但在最终实现时[原提案](https://github.com/golang/proposal/blob/master/design/17503-eliminate-rescan.md)中对 `ptr` 的着色还额外包含对执行栈的着色检查，但由于
时间有限，并未完整实现过，所以混合写屏障在目前（Go 1.14）的实现伪代码是：

```go
// 混合写屏障
func HybridWritePointerSimple(slot *unsafe.Pointer, ptr unsafe.Pointer) {
    shade(*slot)
	shade(ptr)
    *slot = ptr
}
```

在这个实现中，如果无条件对引用双方进行着色，自然结合了 Dijkstra 和 Yuasa 写屏障的优势，
但缺点也非常明显，因为着色成本是双倍的，而且编译器需要插入的代码也成倍增加，随之带来的
结果就是编译后的二进制文件大小也进一步增加。为了针对写屏障的性能进行优化，Go 1.10 前后
Go 团队随后实现了批量写屏障机制。其基本想法是将需要着色的指针同一写入一个缓存，
每当缓存满时统一对缓存中的所有 `ptr` 指针进行着色。

# GC 的实现细节

## 10. Go 语言中 GC 的流程是什么？

当前版本的 Go （1.14）以 STW 为界限，可以将 GC 划分为五个阶段：

|       阶段        |                         说明                         | 赋值器状态 |
| :---------------: | :--------------------------------------------------: | :--------: |
|      GCMark       |    标记准备阶段，为并发标记做准备工作，启动写屏障    |    STW     |
|      GCMark       |      扫描标记阶段，与赋值器并发执行，写屏障开启      |    并发    |
| GCMarkTermination | 标记终止阶段，保证一个周期内标记任务完成，停止写屏障 |    STW     |
|       GCoff       | 内存清扫阶段，将需要回收的内存归还到堆中，写屏障关闭 |    并发    |
|       GCoff       | 内存归还阶段，将过多的内存归还给操作系统，写屏障关闭 |    并发    |

具体而言，各个阶段的触发函数分别为：

![gc-process](https://user-images.githubusercontent.com/7698088/71047691-930aba00-2177-11ea-84d5-4e9eac2df723.png)

## 11. 触发 GC 的时机是什么？

Go 语言中对 GC 的触发时机存在两种形式：  

1. **主动触发**，通过调用 runtime.GC 来触发 GC，此调用阻塞式的等待当前 GC 运行完毕。

2. **被动触发**，分为两种方式：

   - 使用系统监控， 当超过两分钟没有产生任何 GC 时，强制触发 GC。

   - 使用步调（Pacing）算法，其核心思想是控制内存增长的比例。

通过 `GOGC` 或者 `debug.SetGCPercent` 进行控制（他们控制的是同一个变量，即堆的增长率 $\rho$）。整个算法的设计考虑的是优化问题：如果设上一次 GC 完成时，内存的数量为 $H_m$（heap marked），估计需要触发 GC 时的堆大小 $H_T$（heap trigger），使得完成 GC 时候的目标堆大小 $H_g$（heap goal） 与实际完成时候的堆大小 $H_a$（heap actual）最为接近，即 $\min |H_g - H_a| = \min|(1+\rho)H_m - H_a|$

![gc-pacing](https://user-images.githubusercontent.com/7698088/71047935-5e4b3280-2178-11ea-9abd-c86667ac9f88.png)

除此之外，步调算法还需要考虑 CPU 利用率的问题，显然我们不应该让垃圾回收器占用过多的 CPU 使用，即不应该让每个负责执行用户 goroutine 的线程都在执行标记过程，理想情况下，在用户代码满载的情况下，GC 的 CPU 使用率不应该超过 25%，即另一个优化问题：如果设 $u_g$为目标 CPU 使用率（goal utilization），而 $u_a$为实际 CPU 使用率（actual utilization），则 $\min|u_g - u_a|$。

> 求解这两个优化问题的具体数学建模过程我们不在此做深入讨论，有兴趣的读者可以参考两个设计文档：[Go 1.5 concurrent garbage collector pacing](https://docs.google.com/document/d/1wmjrocXIWTr1JxU-3EQBI6BK6KgtiFArkG47XK73xIQ/edit#) 和 [Separate soft and hard heap size goal](https://github.com/golang/proposal/blob/master/design/14951-soft-heap-limit.md)。

计算 $H_T$ 的最终结论（从 Go 1.10 时开始 $h_t$ 增加了上界 $0.95 \rho$，从 Go 1.14 开始时 $h_t$ 增加了下界 0.6）是：

- 设第 n 次触发 GC 时 (n > 1)，估计得到的堆增长率为 $h_t^{(n)}$、运行过程中的实际堆增长率为 $h_a^{(n)}$，用户设置的增长率为 $\rho = \text{GOGC}/100$（ $\rho > 0$）则第 $n+1$ 次出触发 GC 时候，估计的堆增长率为：

$$
h_t^{(n+1)} = h_t^{(n)} + 0.5 \left[ \frac{H_g^{(n)} - H_a^{(n)}}{H_a^{(n)}} - h_t^{(n)} - \frac{u_a^{(n)}}{u_g^{(n)}} \left( h_a^{(n)} - h_t^{(n)} \right) \right]
$$

- 特别的，$h_t^{(1)} = 7 / 8$，$u_a^{(1)} = 0.25$，$u_g^{(1)} = 0.3$。第一次触发 GC 时，如果当前的堆小于 $4\rho$ MB，则强制调整到 $4\rho$ MB 时触发 GC

- 特别的，当 $h_t^{(n)}<0.6$时，将其调整为 $0.6$，当 $h_t^{(n)} > 0.95 \rho$ 时，将其设置为 $0.95 \rho$

- 默认情况下，$\rho = 1$（即 GOGC = 100），第一次触发 GC 时强制设置触发第一次 GC 为 4MB，可以写如下程序进行验证：

```go
package main

import (
	"os"
	"runtime"
	"runtime/trace"
	"sync/atomic"
)

var stop uint64

// 通过对象 P 的释放状态，来确定 GC 是否已经完成
func gcfinished() *int {
	p := 1
	runtime.SetFinalizer(&p, func(_ *int) {
		println("gc finished")
		atomic.StoreUint64(&stop, 1) // 通知停止分配
	})
	return &p
}

func allocate() {
	// 每次调用分配 0.25MB
	_ = make([]byte, int((1<<20)*0.25))
}

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

	gcfinished()

	// 当完成 GC 时停止分配
	for n := 1; atomic.LoadUint64(&stop) != 1; n++ {
		println("#allocate: ", n)
		allocate()
	}
	println("terminate")
}
```

我们先来验证最简单的一种情况，即第一次触发 GC 时的堆大小：

```
$ go1.14 build -o main
$ GODEBUG=gctrace=1 ./main
#allocate:  1
(...)
#allocate:  20
gc finished
gc 1 @0.001s 3%: 0.016+0.23+0.019 ms clock, 0.20+0.11/0.060/0.13+0.22 ms cpu, 4->5->1 MB, 5 MB goal, 12 P
scvg: 8 KB released
scvg: inuse: 1, idle: 62, sys: 63, released: 58, consumed: 5 (MB)
terminate
```

通过这一行数据我们可以看到：

```
gc 1 @0.001s 3%: 0.016+0.23+0.019 ms clock, 0.20+0.11/0.060/0.13+0.22 ms cpu, 4->5->1 MB, 5 MB goal, 12 P
```

1. 程序在完成第一次 GC 后便终止了程序，符合我们的设想
2. 第一次 GC 开始时的堆大小为 4MB，符合我们的设想
3. 当标记终止时，堆大小为 5MB，此后开始执行清扫，这时分配执行到第 20 次，即 20*0.25 = 5MB，符合我们的设想

我们将分配次数调整到 50 次

```go
for n := 1; n < 50; n++ {
	println("#allocate: ", n)
	allocate()
}
```

来验证第二次 GC 触发时是否满足公式所计算得到的值（为 GODEBUG 进一步设置 `gcpacertrace=1`）：

```
$ go1.14 build -o main
$ GODEBUG=gctrace=1,gcpacertrace=1 ./main
#allocate:  1
(...)

pacer: H_m_prev=2236962 h_t=+8.750000e-001 H_T=4194304 h_a=+2.387451e+000 H_a=7577600 h_g=+1.442627e+000 H_g=5464064 u_a=+2.652227e-001 u_g=+3.000000e-001 W_a=152832 goalΔ=+5.676271e-001 actualΔ=+1.512451e+000 u_a/u_g=+8.840755e-001
#allocate:  28
gc 1 @0.001s 5%: 0.032+0.32+0.055 ms clock, 0.38+0.068/0.053/0.11+0.67 ms cpu, 4->7->3 MB, 5 MB goal, 12 P

(...)
#allocate:  37
pacer: H_m_prev=3307736 h_t=+6.000000e-001 H_T=5292377 h_a=+7.949171e-001 H_a=5937112 h_g=+1.000000e+000 H_g=6615472 u_a=+2.658428e-001 u_g=+3.000000e-001 W_a=154240 goalΔ=+4.000000e-001 actualΔ=+1.949171e-001 u_a/u_g=+8.861428e-001
#allocate:  38
gc 2 @0.002s 9%: 0.017+0.26+0.16 ms clock, 0.20+0.079/0.058/0.12+1.9 ms cpu, 5->5->0 MB, 6 MB goal, 12 P
```

我们可以得到数据：

- 第一次估计得到的堆增长率为 $h_t^{(1)} = 0.875$
- 第一次的运行过程中的实际堆增长率为 $h_a^{(1)} = 0.2387451$
- 第一次实际的堆大小为 $H_a^{(1)}=7577600$
- 第一次目标的堆大小为 $H_g^{(1)}=5464064$
- 第一次的 CPU 实际使用率为 $u_a^{(1)} = 0.2652227$
- 第一次的 CPU 目标使用率为 $u_g^{(1)} = 0.3$

我们据此计算第二次估计的堆增长率：

$$
\begin{align}
h_t^{(2)} &= h_t^{(1)} + 0.5 \left[ \frac{H_g^{(1)} - H_a^{(1)}}{H_a^{(1)}} - h_t^{(1)} - \frac{u_a^{(1)}}{u_g^{(1)}} \left( h_a^{(1)} - h_t^{(1)} \right) \right] \\
&= 0.875 + 0.5 \left[ \frac{5464064 - 7577600}{5464064} - 0.875 - \frac{0.2652227}{0.3} \left( 0.2387451 - 0.875 \right) \right] \\
& \approx 0.52534543909 \\
\end{align}
$$

因为 $0.52534543909 < 0.6\rho = 0.6$，因此下一次的触发率为 $h_t^{2} = 0.6$，与我们实际观察到的第二次 GC 的触发率 0.6 吻合。

## 12. 如果内存分配速度超过了标记清除的速度怎么办？是怎么实现的？

目前的 Go 实现中，当 GC 触发后，会首先进入并发标记的阶段。并发标记会设置一个标志，并在 mallocgc 调用进行检查。当存在新的内存分配时，会暂停分配内存过快的那些 goroutine，并将其转去执行一些辅助标记（Mark Assist）的工作，从而达到放缓继续分配、辅助 GC 的标记工作的目的。

编译器会分析用户代码，并在需要分配内存的位置，将申请内存的操作翻译为 `mallocgc` 调用，而 `mallocgc` 的实现决定了标记辅助的实现，其伪代码思路如下：

```go
func mallocgc(t typ.Type, size uint64) {
	if enableMarkAssist {
		// 进行标记辅助，此时用户代码没有得到执行
		(...)
	}
	// 执行内存分配
	(...)
}
```

# GC 的优化问题

## 13. GC 关注的指标有哪些？

Go 中的 GC 被设计为成比例触发、大部分工作与赋值器并发、不分代、无内存移动且会主动向操作系统归还申请的内存。因此最主要关注的能够影响赋值器的性能指标有：

- CPU 利用率：回收算法会在多大程度上拖慢程序？有时候，这个是通过回收占用的 CPU 时间与其它 CPU 时间的百分比来描述的。
- GC 停顿时间：回收器会造成多长时间的停顿？目前的 GC 中需要考虑 STW 和 Mark Assist 两个部分可能造成的停顿。
- GC 停顿频率：回收器造成的停顿频率是怎样的？目前的 GC 中需要考虑 STW 和 Mark Assist 两个部分可能造成的停顿。
- GC 可扩展性：当堆内存变大时，垃圾回收器的性能如何？但大部分的程序可能并不一定关心这个问题。

## 14. Go 的 GC 如何调优？

GC 的调优是在特定场景下产生的，并非所有程序都需要针对 GC 进行调优，只有那些对执行延迟非常敏感、
当 GC 的开销成为程序性能瓶颈的程序，才需要针对 GC 进行性能调优。总的来说，有以下几种情况：

1. 对时间敏感的流式应用：GC 产生的长时间停顿，导致需要立即执行的用户代码执行滞后
2. 需要频繁对同一类对象进行内存分配的应用：频繁分配内存增加 GC 的工作量，影响用户代码的对 CPU 的利用率

所谓 GC 调优的核心思想是：优化内存的申请速度，尽可能的少申请内存，复用已申请的内存（例如 sync.Pool）。通过我们前面介绍的四种观察 GC 的方式，能够快速帮助我们定位存在大量内存申请的位置。
我们将通过三个实际例子，并主要使用第两种（即 `go tool pprof` 和 `go tool trace`）方式来介绍如何定位 GC 的存在的问题，并一步一步进行性能调优。

### 例1

TODO:

### 例2

TODO:

### 例3

TODO:

我们来总结以下前面三个例子中的优化情况：

1. 减少返回会逃逸到堆上分配的变量，是变量作为局部变量进行分配，从而可以直接在不被垃圾回收进行清理的情况下，使整个栈供下一个 goroutine 复用。
2. 将小对象整理为一个大对象来避免内存碎片的问题。
3. 使用 sync.Pool 来复用需要频繁创建临时对象。
4. 使用 goroutine 池，避免短时间内大量创建 goroutine 导致的不可逆的性能下降。

## 15. Go 的垃圾回收器有哪些相关的 API？其作用分别是什么？

在 Go 中存在数量极少的与 GC 相关的 API，它们是

- runtime.GC：手动触发 GC
- runtime.ReadMemStats：读取内存相关的统计信息，其中有包含部分 GC 相关的统计信息
- debug.FreeOSMemory：手动将内存归还给操作系统
- debug.ReadGCStats：读取关于 GC 的相关统计信息
- debug.SetGCPercent：设置 GOGC 调步变量
- debug.SetMaxHeap（尚未发布）：设置 Go 程序堆的上限值
  <!-- - https://github.com/golang/go/issues/23044 -->

# GC 的历史及演进

## 16. Go 历史各个版本在 GC 方面的改进？

- Go 1：串行三色标记清扫

- Go 1.3：并行清扫，标记过程需要 STW，停顿时间在约几百毫秒

- Go 1.5：并发标记清扫，停顿时间在一百毫秒以内

![https://twitter.com/brianhatfield/status/634166123605331968?lang=en](./assets/1.png)

- Go 1.6：使用 bitmap 来记录回收内存的位置，大幅优化垃圾回收器自身消耗的内存，停顿时间在十毫秒以内

  - 从 1.5 切换到 1.6 时的停顿时间对比 
    ![https://twitter.com/brianhatfield/status/692778741567721473/photo/1](./assets/2.jpeg)
  - 从 1.6 切换到 1.6.3 时的停顿时间对比 
    ![https://twitter.com/brianhatfield/status/767812346760859648/photo/1](./assets/3.jpeg)
  - 包括 CPU 使用率的一些改进 
    ![https://twitter.com/brianhatfield/status/768788316347068416/photo/1](./assets/4.jpeg)

- Go 1.7：停顿时间控制在两毫秒以内

  - 从 1.6.3 切换到 1.7 时的停顿时间对比
    ![https://twitter.com/brianhatfield/status/767890817587539968/photo/1](./assets/5.jpeg)

- Go 1.8：混合写屏障，停顿时间在半个毫秒左右

  - ![https://twitter.com/brianhatfield/status/804355831080751104/photo/1](./assets/6.jpeg)

- Go 1.9：彻底移除了栈的重扫描过程

  - 消除栈重扫，与 1.8.3 几乎一致 ![https://twitter.com/brianhatfield/status/900473287750365184/photo/1](./assets/7.jpeg)

- Go 1.12：整合了两个阶段的 Mark Termination，但引入了一个严重的 GC Bug 至今未修，尚无该 Bug 对 GC 性能影响的报告

- Go 1.13：着手解决向操作系统归还内存的，提出了新的 Scavenger

- Go 1.14：替代了仅存在了一个版本的 scavenger，全新的页分配器，优化分配内存过程的速率与现有的扩展性问题，并引入了异步抢占，解决了由于密集循环导致的 STW 时间过长的问题。

可以用下图直观说明 GC 的演进历史：

![](assets/gc1.png)

在 Go 1 刚发布时的版本中，甚至没有将 Mark-Sweep 的过程并行化，当需要进行垃圾回收时，所有的代码都必须进入 STW 的状态。而到了 Go 1.1 时，官方迅速的将清扫过程进行了并行化的处理，即仅在标记阶段进入 STW。这一想法很自然，因为并行化导致算法结果不一致的情况仅仅发生在标记阶段，而当时的垃圾回收器没有针对并行结果的一致性进行任何优化，因此才需要在标记阶段进入 STW。对于 Scavenger 而言，早起的版本中会有一个单独的线程来定期将多余的内存归还给操作系统。

![](assets/gc2.png)

而到了 Go 1.5 后，Go 团队花费了相当大的力气，通过引入写屏障的机制来保证算法的一致性，才得以将整个 GC 控制在很小的 STW 内，而到了 1.8 时，由于新的混合屏障的出现，消除了对栈本身的重新扫描 STW 的时间进一步得以缩减。从这个时候开始，Scavenger 已经从独立线程中移除，并合并至系统监控这个独立的线程中，并周期性的向操作系统归还内存，但仍然会有内存溢出这种比较极端的情况出现，因为程序可能在短时间内应对突发性的内存申请需求时，内存还没来得及归还操作系统，导致堆不断向操作系统申请内存，从而出现内存溢出。

![](assets/gc3.png)

到了 Go 1.13，定期归还操作系统的问题得以解决，Go 团队开始将周期性的 Scavenger 转化为可被调度的 goroutine，并将其与用户代码并发执行。而到了 Go 1.14，这一向操作系统归还内存的操作时间进一步得到缩减。

## 17. Go GC 在演化过程中还存在那些其他设计？为什么他们没有被接受？

### 并发栈重扫

正如我们前面所说，允许灰色赋值器存在的垃圾回收器需要引入重扫过程来保证算法的正确性，
除了引入混合屏障来消除重扫这一过程外，有另一种做法可以提高重扫过程的性能，那就是将重扫
的过程并发执行。然而这一[方案](https://github.com/golang/proposal/blob/master/design/17505-concurrent-rescan.md)并没有得以实现，原因很简单：实现过程相比引入混合屏障而言十分复杂，
而且引入混合屏障能够消除重扫这一过程将简化垃圾回收的步骤。

### ROC

ROC 的全称叫[面向请求的回收器](https://docs.google.com/document/d/1gCsFxXamW8RRvOe5hECz98Ftk-tcRRJcDFANj2VwCB0/edit)（Request Oriented Collector），它
其实也是分代 GC 的一种重新叙述。它提出了一个
请求假设（Request Hypothesis）：与一个完整请求、休眠 goroutine 所关联的对象比其他对象更容易死亡。
这个假设听起来非常符合直觉，但在实现上，由于垃圾回收器必须确保是否有 goroutine 私有
指针被写入公共对象，因此写屏障必须一直打开，这也就产生了该方法的致命缺点：
昂贵的写屏障及其带来的导致缓存未命中，这也是这一设计最终没有被接受的主要原因。

### 传统分代 GC

在发现 ROC 性能不行之后，作为备选方案，Go 团队还尝试了实现[传统的分代式 GC](https://go-review.googlesource.com/c/go/+/137476/12)。但最终同样发现分代假设并不适用于 
Go 的运行栈机制，年轻代对象在栈上就已经死亡，扫描本就该回收的执行栈并没有为由于分代假设带来明显的性能提升。
这也是这一设计最终没有被接受的主要原因。

## 18. 目前提供 GC 的语言以及不提供 GC 的语言有哪些？GC 和 No GC 各自的优缺点是什么？

从原理上而言，所有的语言都能够自行实现 GC。从语言诞生之初就提供 GC 的语言，例如：

- Python
- JavaScript
- Java
- Objective-C
- Swift

而不以 GC 为目标，被直接设计为手动管理内存、但可以自行实现 GC 的语言有：

- C
- C++

也有一些语言可以在编译期，依靠编译器插入清理代码的方式，实现精准的清理，例如 

- Rust

垃圾回收使程序员无需手动处理内存释放，从而能够消除一些需要手动管理内存才会出现的运行时错误：

1. 在仍然有指向内存区块的指针的情况下释放这块内存时，会产生悬挂指针，从而后续可能错误的访问已经用做他用的内存区域。
2. 多重释放同一块申请的内存区域可能导致不可知的内存损坏。

当然，垃圾回收也会伴随一些缺陷，这也就造就了没有 GC 的一些优势：

1. 没有额外的性能开销
2. 精准的手动内存管理，极致的利用机器的性能

## 19. Go 对比 Java、V8 中 JavaScript 的 GC 性能如何？

无论是 Java 还是 JavaScript 中的 GC 均为分代式 GC。分代式 GC 的一个核心假设就是分代假说：将对象依据存活时间分配到不同的区域，每次回收只回收其中的一个区域。

### V8 的 GC

在V8中主要将内存分为新生代和老生代。新生代中的对象为存活时间较短的对象，老生代中的对象为存活时间较长或常驻内存、或者占用内存较大的对象：

1. 新生代中的对象主要通过副垃圾回收器进行回收。该回收过程是一种采用复制的方式实现的垃圾回收算法，它将堆内存一分为二，这两个空间中只有一个处于使用中，一个处于闲置状态。处于是使用状态的空间称为From空间，处于闲置的空间称为To空间。分配对象时，先是在From空间中进行分配，当开始垃圾回收时，会检查From空间中的存活对象，并将这些存活对象复制到To空间中，而非存活对象占用的空间被释放。完成复制后，From空间和To空间的角色互换。即就是通过将存活对象在两个空间中进行复制。
2. 老生代则由主垃圾回收器负责。它的实现的是标记清扫过程，但略有不同之处在于它还会在清扫完成后对内存碎片进行整理，进而是一种标记整理的回收器。

### Java 的 GC

Java 的 GC 称之为 G1，并将整个堆分为年轻代、老年代和永久代。包括四种不同的收集操作，
根据代码行为从上往下依次选择性与配置选择性进行：

1. 年轻代收集周期：只对年轻代对象进行收集与清理
2. 老年代收集周期：只对老年代对象进行收集与清理
3. 混合式收集周期：同时对年轻代和老年代进行收集与清理
4. 完整 GC 周期：完整的对整个堆进行收集与清理

在回收过程中，G1 会对停顿时间进行预测，竭尽所能的调整 GC 的策略从而达到用户代码
通过系统参数（`-XX:MaxGCPauseMillis`）所配置的对停顿时间的要求。
这四个周期的执行成本逐渐上升，优化得当的程序可以完全避免完整 GC 周期。

### 性能比较

在 Go、Java 和 V8 JavaScript 之间比较 GC 的性能本质上是一个不切实际的问题。如前面所说，垃圾回收器的设计权衡了相当多方面的因素，除此之外同时还受语言自身设计的影响，因为语言的设计也直接影响了程序员编写代码的形式，也就自然影响了产生垃圾的方式。
但总的来说，他们三者对垃圾回收的实现都需要 STW，并均已达到了用户代码几乎无法感知到的状态（根 GC 作者 Austin 宣称 [STW 小于 100 微秒](https://groups.google.com/d/msg/golang-dev/Ab1sFeoZg_8/_DaL0E8fAwAJ)）。当然，随着 STW 的减少，垃圾回收器会增加 CPU 的使用率，这也是程序员在编写代码时需要手动进行优化的部分，即充分考虑内存分配的必要性，减少过多申请内存带给垃圾回收器的压力。

## 20. 目前 Go 语言的 GC 还存在哪些问题？

尽管 Go 团队宣称 STW 停顿时间得以优化到 100 微秒级别，但这本质上是一种取舍。
原本的 STW 某种意义上来说其实转移到了可能导致用户代码停顿的几个位置；
除此之外，由于运行时调度器的实现方式，同样对 GC 存在一定程度的影响。
目前（Go 1.14）Go 中的 GC 仍然存在以下问题：

### 1. Mark Assist 停顿时间过长

```go
package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/trace"
	"time"
)

const (
	windowSize = 200000
	msgCount   = 1000000
)

var (
	best    time.Duration = time.Second
	bestAt  time.Time
	worst   time.Duration
	worstAt time.Time

	start = time.Now()
)

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

	for i := 0; i < 5; i++ {
		measure()
		worst = 0
		best = time.Second
		runtime.GC()
	}
}

func measure() {
	var c channel
	for i := 0; i < msgCount; i++ {
		c.sendMsg(i)
	}
	fmt.Printf("Best send delay %v at %v, worst send delay: %v at %v. Wall clock: %v \n", best, bestAt.Sub(start), worst, worstAt.Sub(start), time.Since(start))
}

type channel [windowSize][]byte

func (c *channel) sendMsg(id int) {
	start := time.Now()

	// 模拟发送
	(*c)[id%windowSize] = newMsg(id)

	end := time.Now()
	elapsed := end.Sub(start)
	if elapsed > worst {
		worst = elapsed
		worstAt = end
	}
	if elapsed < best {
		best = elapsed
		bestAt = end
	}
}

func newMsg(n int) []byte {
	m := make([]byte, 1024)
	for i := range m {
		m[i] = byte(n)
	}
	return m
}
```

通过运行此程序我们可以得到类似下面的结果：

```
$ go1.14 run main.go

Best send delay 330ns at 773.037956ms, worst send delay: 7.127915ms at 579.835487ms. Wall clock: 831.066632ms 
Best send delay 331ns at 873.672966ms, worst send delay: 6.731947ms at 1.023969626s. Wall clock: 1.515295559s 
Best send delay 330ns at 1.812141567s, worst send delay: 5.34028ms at 2.193858359s. Wall clock: 2.199921749s 
Best send delay 338ns at 2.722161771s, worst send delay: 7.479482ms at 2.665355216s. Wall clock: 2.920174197s 
Best send delay 337ns at 3.173649445s, worst send delay: 6.989577ms at 3.361716121s. Wall clock: 3.615079348s 
```
![](./assets/gc-mark-assist.png)

在这个结果中，第一次的最坏延迟时间高达 7.12 毫秒，发生在程序运行 578 毫秒左右。通过 `go tool trace` 可以发现，这个时间段中，Mark Assist 执行了 7112312ns，约为 7.127915ms；可见，此时最坏情况下，标记辅助拖慢了用户代码的执行，是造成 7 毫秒延迟的原因。

<!-- https://github.com/golang/go/issues/27732 -->
<!-- https://github.com/golang/go/issues/27410 -->

### 2. Sweep 停顿时间过长

同样还是刚才的例子，如果我们仔细观察 Mark Assist 后发生的 Sweep 阶段，竟然对用户代码的影响长达约 30ms，根据调用栈信息可以看到，该 Sweep 过程发生在内存分配阶段：

![](./assets/gc-mark-sweep.png)

<!-- https://github.com/golang/go/issues/18155 -->

### 3. 由于 GC 算法的不正确性导致 GC 周期被迫重新执行

此问题很难复现，但是一个已知的问题，根据 Go 团队的描述，
[能够在 1334 次构建中发生一次](https://github.com/golang/go/issues/27993#issuecomment-441719687)，我们可以计算出其触发概率约为 0.0007496251874。虽然发生概率很低，但一旦发生，GC 需要被重新执行，非常不幸。

<!-- https://github.com/golang/go/issues/27993 -->

### 4. 创建大量 Goroutine 后导致 GC 消耗更多的 CPU

这个问题可以通过以下程序进行验证：

```go
func BenchmarkGCLargeGs(b *testing.B) {
	wg := sync.WaitGroup{}

	for ng := 100; ng <= 1000000; ng *= 10 {
		b.Run(fmt.Sprintf("#g-%d", ng), func(b *testing.B) {
			// 创建大量 goroutine，由于每次创建的 goroutine 会休眠
			// 从而运行时不会复用正在休眠的 goroutine，进而不断创建新的 g
			wg.Add(ng)
			for i := 0; i < ng; i++ {
				go func() {
					time.Sleep(100 * time.Millisecond)
					wg.Done()
				}()
			}
			wg.Wait()

			// 现运行一次 GC 来提供一致的内存环境
			runtime.GC()

			// 记录运行 b.N 次 GC 需要的时间
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				runtime.GC()
			}
		})

	}
}
```

其结果可以通过如下指令来获得：

```shell
$ go1.14 test -bench=BenchmarkGCLargeGs -run=^$ -count=5 -v . | tee 4.txt
$ benchstat 4.txt
name                     time/op
GCLargeGs/#g-100-12       192µs ± 5%
GCLargeGs/#g-1000-12      331µs ± 1%
GCLargeGs/#g-10000-12    1.22ms ± 1%
GCLargeGs/#g-100000-12   10.9ms ± 3%
GCLargeGs/#g-1000000-12  32.5ms ± 4%
```

这种情况通常发生于峰值流量后，大量 goroutine 由于任务等待被休眠，从而运行时不断创建新的 goroutine，
旧的 goroutine 由于休眠未被销毁且得不到复用，导致 GC 需要扫描的执行栈越来越多，进而完成 GC 所需的时间越来越长。
一个解决办法是使用 Goroutine 池来限制创建的 goroutine 数量。

<!-- https://github.com/golang/go/issues/34457 -->

### 5. 当堆较小时，GC 可能导致内存页抖动

TODO:

<!-- https://github.com/golang/go/issues/22743 -->

### 6. 结构体的对齐要求导致内存浪费，从而增加 GC 的开销

TODO:

<!-- https://github.com/golang/go/issues/8885 -->
<!-- https://github.com/golang/go/issues/11979 -->

### 7. GC 导致的内存碎片

<!-- https://github.com/golang/go/issues/21695 -->

# 总结

GC 是一个复杂的系统工程，本文讨论的二十个问题尽管已经展现了一个相对全面的 Go GC。
但它们仍然只是 GC 这一宏观问题的一些较为重要的部分，还有非常多的细枝末节、研究进展
无法在有限的篇幅内完整讨论。
从 Go 诞生之初，Go 团队就一直在对 GC 的表现进行实验与优化，但仍然有诸多未解决的问题，
我们不妨对 GC 未来的改进拭目以待。