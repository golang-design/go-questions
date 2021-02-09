---
weight: 212
title: "map 是线程安全的吗"
slug: /thread-safety
---

map 不是线程安全的。

在查找、赋值、遍历、删除的过程中都会检测写标志，一旦发现写标志置位（等于1），则直接 panic。赋值和删除函数在检测完写标志是复位之后，先将写标志位置位，才会进行之后的操作。

检测写标志：

```golang
if h.flags&hashWriting == 0 {
		throw("concurrent map writes")
	}
```

设置写标志：

```golang
h.flags |= hashWriting
```
