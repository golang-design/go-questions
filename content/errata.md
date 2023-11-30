---
title: 勘误表
---

# 《Go 程序员面试笔试宝典》勘误表

请直接搜索对应的页面，如“第 255 页”，查看该页面的勘误内容。

## 第 V 页
- typo
4.3 通道的底结构 -> 通道的`底层`结构。

## 第 9 页

- 多了一个逗号

<img width="928" alt="image" src="https://user-images.githubusercontent.com/7698088/168478109-96352e26-48f6-4001-9a1d-4be4fb63fae8.png">


- 少了一个括号

<img width="721" alt="image" src="https://user-images.githubusercontent.com/7698088/167523651-7bfd8efc-a499-4a57-b9e1-7a1a83adb1e5.png">

## 第 13 页

- 调用 deferproc 函数

<img width="909" alt="image" src="https://user-images.githubusercontent.com/7698088/168234622-42fdb525-026d-42a8-af9e-31b92374f9ec.png">


## 第 27 页

- 形参是实参的一个复制

<img width="850" alt="image" src="https://user-images.githubusercontent.com/7698088/170994248-7dbb1f1d-1707-42c5-8ac6-cf532128abfe.png">


## 第 32 页

- s/hamp/hmap

<img width="909" alt="image" src="https://user-images.githubusercontent.com/5498964/168425490-826e0972-c550-4892-9a23-f64acdbd1aa4.png">

## 第 35 页

- 返回 value 类型的零值

<img width="946" alt="image" src="https://user-images.githubusercontent.com/7698088/168294619-a14a96a5-a61b-4169-8c53-5f205ee075f9.png">


## 第 73 页

- 示例代码和运行结果修正

示例代码：

```golang
type Student struct {
	name string
	age  int
}

var s = Student{name: "qcrao", age: 18}
var g = &s

func modifyUser(pu *Student) {
	fmt.Println("modifyUser received value:", pu)
	pu.name = "Old Old qcrao"
	pu.age = 200
}

func printUser(u <-chan *Student) {
	time.Sleep(2 * time.Second)
	fmt.Println("printUser get:", <-u)
}

func main() {
	c := make(chan *Student, 5)
	c <- g
	fmt.Println(g)
	// modify g
	g = &Student{name: "Old qcrao", age: 100}
	go printUser(c)
	go modifyUser(g)
	time.Sleep(5 * time.Second)
	fmt.Println(g)
}

```

运行结果：

```shell
&{qcrao 18}
modifyUser received value: &{Old qcrao 100}
printUser get: &{qcrao 18}
&{Old Old qcrao 200}
```

- 图中变量标注有误：

<img width="822" alt="image" src="https://user-images.githubusercontent.com/5498964/168426034-165dfb6b-5142-43df-850e-063d7a18b574.png">


## 第 76 页

- IsClosed 函数返回 false

<img width="1187" alt="image" src="https://user-images.githubusercontent.com/7698088/165060080-fc5d069f-35f3-4e4e-aaf9-f2c359ea65a6.png">

## 第 89 页

- 多余的“个”

<img width="976" alt="image" src="https://user-images.githubusercontent.com/7698088/168477280-a32572ce-59ca-4bef-a4b8-dabf29c09447.png">


## 第 112 页

- 当 n==5 时，描述有问题

<img width="924" alt="image" src="https://user-images.githubusercontent.com/7698088/167293610-c62635e2-e52a-470c-b9fd-e7ca56331581.png">

## 第 113 页

- context 包代码结构功能表格描述有问题

<img width="632" alt="image" src="https://user-images.githubusercontent.com/7698088/167293763-781c293e-7a0c-4f72-b083-1356cf1e5afb.png">

## 第 122 页

- 多了一个 String 方法的代码

<img width="1147" alt="image" src="https://user-images.githubusercontent.com/7698088/226241857-602b4d87-ade1-4bdc-a973-9d7ece991ec6.png">


## 第 131 页

- 去掉多余的描述

<img width="1000" alt="image" src="https://user-images.githubusercontent.com/7698088/175322427-13da90eb-700a-45d9-8a20-3c15f8ae4ba5.png">

## 第 132 页

- ![](https://raw.githubusercontent.com/qcrao/blog/master/pics20231130182154.png)


## 第 141 页

- 有关 interface 的章节说明有误

<img width="744" alt="image" src="https://user-images.githubusercontent.com/7698088/174421449-8e81b69d-95db-4472-9c6b-e5bb7f1a11a5.png">


## 第 174 页

- 图标注有误

<img width="964" alt="image" src="https://user-images.githubusercontent.com/7698088/168816746-59a6683c-c6e5-4a83-951d-13ff2a25aedb.png">


## 第 186 页

- 关于 M 的描述有误：

<img width="1025" alt="image" src="https://user-images.githubusercontent.com/7698088/168823052-85522c94-5f77-4f99-bb56-c5bbb44cb679.png">

上面红框内的这两段话替换为：

> Go 调度循环可以看成是一个“生产-消费”的流程。

> 生产端就是我们写的 go func()...语句，它会产生一个 goroutine。消费者是 M，所有的 M 都是在不断地执行调度循环：找到 runnable 的 goroutine 来运行，运行完了就去找下一个 goroutine……

> P 的个数是固定的，它等于 GOMAXPROCS 个，进程启动的时候就会被全部创建出来。随着程序的运行，越来越多的 goroutine 会被创建出来。这时，M 也会随之被创建，用于执行 goroutine，M 的个数没有一定的规律，视 goroutine 情况而定。

- 多了个“不”字

<img width="1117" alt="image" src="https://user-images.githubusercontent.com/7698088/226796514-82eab03f-7214-4a4a-9584-306e01ca8072.png">


# 第 191 页

- memory 拼写错误

<img width="779" alt="image" src="https://user-images.githubusercontent.com/7698088/178105292-5737eb90-d2b0-43f5-8c1b-5558a5116aca.png">


## 第 192 页

- s/_Grunnale/_Grunnable

<img width="632" alt="image" src="https://user-images.githubusercontent.com/5498964/168425331-2518fbb8-4b6b-4a40-ba69-b2e8e91659ac.png">

## 第 211 页

- 图 12-19 m0 标识错误

<img width="768" alt="image" src="https://user-images.githubusercontent.com/7698088/167159386-d56ae249-b1fe-4b80-87a6-a7134031c27e.png">

# 第 235 页

- notewakeup 唤醒

<img width="1169" alt="image" src="https://user-images.githubusercontent.com/7698088/172192016-78934c54-8b1c-42ed-8349-64582bc0ca93.png">


## 第 240 页

- 12.14.1 小节中，"这时和P绑定的G正在进行系统调用，无法执行其他的G" 更改为 "这时和P绑定的M正在进行系统调用，无法执行其他的G"

## 第 255 页

- 图 13-3 标识错误，如下图

<img width="552" alt="image" src="https://user-images.githubusercontent.com/7698088/164480491-2322a633-8fdf-41fa-9fa0-06ec82680188.png">

## 第 258 页

- 表 13-1 不同等级的浪费”下方算式错误

<img width="868" alt="image" src="https://user-images.githubusercontent.com/7698088/164685097-518683e5-8a8b-4f6b-9e04-5f2952d8cea9.png">

## 第 298 页

- GC 时间降到 300us

<img width="857" alt="image" src="https://user-images.githubusercontent.com/7698088/169967864-e5531d4f-930e-45b0-b6ea-2cb52c102f12.png">

## 第 306 页

- 图 14-19 标识错误

<img width="717" alt="image" src="https://user-images.githubusercontent.com/7698088/169996067-efecda48-35e6-4b00-8314-5266d68b849f.png">


