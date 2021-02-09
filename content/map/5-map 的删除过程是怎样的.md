---
weight: 205
title: "删除过程"
slug: /delete
---


写操作底层的执行函数是 `mapdelete`：

```golang
func mapdelete(t *maptype, h *hmap, key unsafe.Pointer) 
```

根据 key 类型的不同，删除操作会被优化成更具体的函数：

|key 类型|删除|
|---|---|
|uint32|mapdelete_fast32(t *maptype, h *hmap, key uint32)|
|uint64|mapdelete_fast64(t *maptype, h *hmap, key uint64)|
|string|mapdelete_faststr(t *maptype, h *hmap, ky string)|

当然，我们只关心 `mapdelete` 函数。它首先会检查 h.flags 标志，如果发现写标位是 1，直接 panic，因为这表明有其他协程同时在进行写操作。

计算 key 的哈希，找到落入的 bucket。检查此 map 如果正在扩容的过程中，直接触发一次搬迁操作。

删除操作同样是两层循环，核心还是找到 key 的具体位置。寻找过程都是类似的，在 bucket 中挨个 cell 寻找。

找到对应位置后，对 key 或者 value 进行“清零”操作：

```golang
// 对 key 清零
if t.indirectkey {
	*(*unsafe.Pointer)(k) = nil
} else {
	typedmemclr(t.key, k)
}

// 对 value 清零
if t.indirectvalue {
	*(*unsafe.Pointer)(v) = nil
} else {
	typedmemclr(t.elem, v)
}
```

最后，将 count 值减 1，将对应位置的 tophash 值置成 `Empty`。

这块源码同样比较简单，感兴起直接去看代码。