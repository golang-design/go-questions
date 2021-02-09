---
weight: 309
title: "如何用 interface 实现多态"
slug: /polymorphism
---

`Go` 语言并没有设计诸如虚函数、纯虚函数、继承、多重继承等概念，但它通过接口却非常优雅地支持了面向对象的特性。

多态是一种运行期的行为，它有以下几个特点：

>1. 一种类型具有多种类型的能力
>2. 允许不同的对象对同一消息做出灵活的反应
>3. 以一种通用的方式对待个使用的对象
>4. 非动态语言必须通过继承和接口的方式来实现

看一个实现了多态的代码例子：

```golang
package main

import "fmt"

func main() {
	qcrao := Student{age: 18}
	whatJob(&qcrao)

	growUp(&qcrao)
	fmt.Println(qcrao)

	stefno := Programmer{age: 100}
	whatJob(stefno)

	growUp(stefno)
	fmt.Println(stefno)
}

func whatJob(p Person) {
	p.job()
}

func growUp(p Person) {
	p.growUp()
}

type Person interface {
	job()
	growUp()
}

type Student struct {
	age int
}

func (p Student) job() {
	fmt.Println("I am a student.")
	return
}

func (p *Student) growUp() {
	p.age += 1
	return
}

type Programmer struct {
	age int
}

func (p Programmer) job() {
	fmt.Println("I am a programmer.")
	return
}

func (p Programmer) growUp() {
	// 程序员老得太快 ^_^
	p.age += 10
	return
}
```

代码里先定义了 1 个 `Person` 接口，包含两个函数：

```golang
job()
growUp()
```

然后，又定义了 2 个结构体，`Student` 和 `Programmer`，同时，类型 `*Student`、`Programmer` 实现了 `Person` 接口定义的两个函数。注意，`*Student` 类型实现了接口， `Student` 类型却没有。

之后，我又定义了函数参数是 `Person` 接口的两个函数：

```golang
func whatJob(p Person)
func growUp(p Person)
```

`main` 函数里先生成 `Student` 和 `Programmer` 的对象，再将它们分别传入到函数 `whatJob` 和 `growUp`。函数中，直接调用接口函数，实际执行的时候是看最终传入的实体类型是什么，调用的是实体类型实现的函数。于是，不同对象针对同一消息就有多种表现，`多态`就实现了。

更深入一点来说的话，在函数 `whatJob()` 或者 `growUp()` 内部，接口 `person` 绑定了实体类型 `*Student` 或者 `Programmer`。根据前面分析的 `iface` 源码，这里会直接调用 `fun` 里保存的函数，类似于： `s.tab->fun[0]`，而因为 `fun` 数组里保存的是实体类型实现的函数，所以当函数传入不同的实体类型时，调用的实际上是不同的函数实现，从而实现多态。

运行一下代码：

```shell
I am a student.
{19}
I am a programmer.
{100}
```

# 参考资料
【各种面向对象的名词】https://cyent.github.io/golang/other/oo/

【多态与鸭子类型】https://www.jb51.net/article/116025.htm