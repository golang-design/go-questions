---
weight: 531
title: "Go指针和unsafe.Pointer有什么区别"
slug: /pointers
---

Go 语言的作者之一 Ken Thompson 也是 C 语言的作者。所以，Go 可以看作 C 系语言，它的很多特性都和 C 类似，指针就是其中之一。

然而，Go 语言的指针相比 C 的指针有很多限制。这当然是为了安全考虑，要知道像 Java/Python 这些现代语言，生怕程序员出错，哪有什么指针（这里指的是显式的指针）？更别说像 C/C++ 还需要程序员自己清理“垃圾”。所以对于 Go 来说，有指针已经很不错了，仅管它有很多限制。

相比于 C 语言中指针的灵活，Go 的指针多了一些限制。但这也算是 Go 的成功之处：既可以享受指针带来的便利，又避免了指针的危险性。

限制一：`Go 的指针不能进行数学运算`。

来看一个简单的例子：

```golang
a := 5
p := &a

p++
p = &a + 3
```

上面的代码将不能通过编译，会报编译错误：`invalid operation`，也就是说不能对指针做数学运算。

限制二：`不同类型的指针不能相互转换`。

例如下面这个简短的例子：

```golang
func main() {
	a := int(100)
	var f *float64
	
	f = &a
}
```

也会报编译错误：

```shell
cannot use &a (type *int) as type *float64 in assignment
```

限制三：`不同类型的指针不能使用 == 或 != 比较`。

只有在两个指针类型相同或者可以相互转换的情况下，才可以对两者进行比较。另外，指针可以通过 `==` 和 `!=` 直接和 `nil` 作比较。

限制四：`不同类型的指针变量不能相互赋值`。

这一点同限制三。

unsafe.Pointer 在 unsafe 包：

```golang
type ArbitraryType int

type Pointer *ArbitraryType
```

从命名来看，`Arbitrary` 是任意的意思，也就是说 Pointer 可以指向任意类型，实际上它类似于 C 语言里的 `void*`。

unsafe 包提供了 2 点重要的能力：

> 1. 任何类型的指针和 unsafe.Pointer 可以相互转换。
> 2. uintptr 类型和 unsafe.Pointer 可以相互转换。

![type pointer uintptr](../assets/0.png)

pointer 不能直接进行数学运算，但可以把它转换成 uintptr，对 uintptr 类型进行数学运算，再转换成 pointer 类型。

```golang
// uintptr 是一个整数类型，它足够大，可以存储
type uintptr uintptr
```

还有一点要注意的是，uintptr 并没有指针的语义，意思就是 uintptr 所指向的对象会被 gc 无情地回收。而 unsafe.Pointer 有指针语义，可以保护它所指向的对象在“有用”的时候不会被垃圾回收。

unsafe 包中的几个函数都是在编译期间执行完毕，毕竟，编译器对内存分配这些操作“了然于胸”。在 `/usr/local/go/src/cmd/compile/internal/gc/unsafe.go` 路径下，可以看到编译期间 Go 对 unsafe 包中函数的处理。