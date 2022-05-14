---
title: 勘误表
---

# 《Go 程序员面试笔试宝典》勘误表

请直接搜索对应的页面，如“第 255 页”，查看该页面的勘误内容。

## 第 9 页

- 少了一个括号

<img width="721" alt="image" src="https://user-images.githubusercontent.com/7698088/167523651-7bfd8efc-a499-4a57-b9e1-7a1a83adb1e5.png">

## 第 13 页

- 调用 deferproc 函数

<img width="909" alt="image" src="https://user-images.githubusercontent.com/7698088/168234622-42fdb525-026d-42a8-af9e-31b92374f9ec.png">


## 第 27 页

- 形参是实参的一个复制

<img width="980" alt="image" src="https://user-images.githubusercontent.com/7698088/168206572-7d9a836a-6697-4e21-a8ce-99a944f3f75d.png">

## 第 32 页

- s/hamp/hmap

<img width="909" alt="image" src="https://user-images.githubusercontent.com/5498964/168425490-826e0972-c550-4892-9a23-f64acdbd1aa4.png">

## 第 35 页

- 返回 value 类型的零值

<img width="946" alt="image" src="https://user-images.githubusercontent.com/7698088/168294619-a14a96a5-a61b-4169-8c53-5f205ee075f9.png">


## 第 73 页

第一段代码块为 (https://go.dev/play/p/SrKx07Mo7Ti)：

```go
package main

import (
	"fmt"
	"time"
)

type Student struct {
	name string
	age  int8
}

var s = Student{name: "qcrao", age: 18}
var g = &s

func modifyStudent(pu *Student) {
	fmt.Println("modifyStudent Received Vaule", pu)
	pu.name = "Stefno"
}
func printStudent(u <-chan *Student) {
	time.Sleep(2 * time.Second)
	fmt.Println("printStudent GoRoutine called", <-u)
}
func main() {
	c := make(chan *Student, 5)
	c <- g
	fmt.Println(g)
	// modify g
	g = &Student{name: "Old qcrao", age: 100}
	go printStudent(c)
	go modifyStudent(g)
	time.Sleep(5 * time.Second)
	fmt.Println(g)
}
```

运行结果为

```
&{qcrao 18}
modifyStudent Received Vaule &{Old qcrao 100}
printStudent GoRoutine called &{qcrao 18}
&{Stefno 100}
```

图中变量标注有误：

<img width="822" alt="image" src="https://user-images.githubusercontent.com/5498964/168426034-165dfb6b-5142-43df-850e-063d7a18b574.png">


## 第 76 页

- IsClosed 函数返回 false

<img width="1187" alt="image" src="https://user-images.githubusercontent.com/7698088/165060080-fc5d069f-35f3-4e4e-aaf9-f2c359ea65a6.png">

## 第 112 页

- 当 n==5 时，描述有问题

<img width="924" alt="image" src="https://user-images.githubusercontent.com/7698088/167293610-c62635e2-e52a-470c-b9fd-e7ca56331581.png">

## 第 113 页

- context 包代码结构功能表格描述有问题

<img width="632" alt="image" src="https://user-images.githubusercontent.com/7698088/167293763-781c293e-7a0c-4f72-b083-1356cf1e5afb.png">

## 第 192 页

- s/_Grunnale/_Grunnable

<img width="632" alt="image" src="https://user-images.githubusercontent.com/5498964/168425331-2518fbb8-4b6b-4a40-ba69-b2e8e91659ac.png">

## 第 211 页

- 图 12-19 m0 标识错误

<img width="768" alt="image" src="https://user-images.githubusercontent.com/7698088/167159386-d56ae249-b1fe-4b80-87a6-a7134031c27e.png">


## 第 240 页

12.14.1 小节中，"这时和P绑定的G正在进行系统调用，无法执行其他的G" 更改为 "这时和P绑定的M正在进行系统调用，无法执行其他的G"

## 第 255 页

- 图 13-3 标识错误，如下图

<img width="552" alt="image" src="https://user-images.githubusercontent.com/7698088/164480491-2322a633-8fdf-41fa-9fa0-06ec82680188.png">

## 第 258 页

- 表 13-1 不同等级的浪费”下方算式错误

<img width="868" alt="image" src="https://user-images.githubusercontent.com/7698088/164685097-518683e5-8a8b-4f6b-9e04-5f2952d8cea9.png">
