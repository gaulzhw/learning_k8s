# linux 性能优化

**Memory 篇**

https://time.geekbang.org/column/intro/100020901?tab=catalog



## 1. 内存使用情况

### free

```shell
# 注意不同版本的free输出可能会有所不同
$ free
        total   used    free shared buff/cache available
Mem:  8169348 263524 6875352    668    1030472   7611064
Swap:       0      0       0
```

- total 是总内存大小
- used 是已使用内存大小，包含了共享内存
- free 是未使用内存的大小
- shared 是共享内存的大小
- buff/cache 是缓存和缓冲区的大小
- available 是新进程可用内存的大小，包含未使用内存和可回收内存



### top

```shell
# 按下M切换到内存排序
$ top
...
KiB Mem : 8169348 total, 6871440 free, 267096 used, 1030812 buff/cache
KiB Swap:       0 total,       0 free,      0 used. 7607492 avail Mem

  PID  USER PR NI   VIRT   RES   SHR S %CPU %MEM   TIME+ COMMAND
  430  root 19 -1 122360 35588 23748 S  0.0  0.4 0:32.17 systemd-journal
 1075  root 20  0 771860 22744 11368 S  0.0  0.3 0:38.89 snapd
 1048  root 20  0 170904 17292  9488 S  0.0  0.2 0:00.24 networkd-dispat
    1  root 20  0  78020  9156  6644 S  0.0  0.1 0:22.92 systemd
12376 azure 20  0  76632  7456  6420 S  0.0  0.1 0:00.01 systemd
12374  root 20  0 107984  7312  6304 S  0.0  0.1 0:00.00 sshd
...
```

- VIRT 是进程虚拟内存的大小，只要是进程申请过的内存，即使还没有真正分配物理内存，也会计算在内
- RES 是常驻内存的大小，也就是进程实际使用的物理内存大小，但不包括 Swap 和共享内存
- SHR 是共享内存的大小，比如与其他进程共同使用的共享内存、加载的动态链接库以及程序的代码段等
- %MEM 是进程使用物理内存占系统总内存的百分比

虚拟内存通常并不会全部分配物理内存，每个进程的虚拟内存都比常驻内存大的多。

共享内存 SHR 并不一定是共享的，比如说程序的代码段、非共享的动态链接库也都算在 SHR 里。SHR 也包括了进程间真正共享的内存。在计算多个进程的内存使用时，不要把所有进程的 SHR 直接相加得出结果。



## 2. Buffer & Cache

```shell
# man proc
Buffers %lu
    Relatively temporary storage for raw disk blocks that shouldn't get tremendously large (20MB or so).

Cached %lu
    In-memory cache for files read from the disk (the page cache). Doesn't include SwapCached.
...
SReclaimable %lu (since Linux 2.6.19)
    Part of Slab, that might be reclaimed, such as caches. 

SUnreclaim %lu (since Linux 2.6.19)
    Part of Slab, that cannot be reclaimed on memory pressure.
```



- Buffers 是对原始数据块的临时存储，也就是用来缓存磁盘的数据，通常不会特别大（20MB 左右），内存就可以把分散的写集中起来，统一优化磁盘的写入
- Cached 是从磁盘读取文件的页缓存，也就是用来缓存从文件读取的数据，下次访问文件数据时就可以直接从内存中快速获取而不需要再次访问磁盘
- SReclaimable 是 Slab 的一部分，Slab 包括两部分：可回收部分用 SReclaimalbe 记录；不可回收部分用 SUnreclaim 记录



### 案例

准备工作，清空缓存

```shell
# 清理文件页、目录项、Inodes等各种缓存
$ echo 3 > /proc/sys/vm/drop_caches
```

/proc/sys/vm/drop_caches 就是通过 proc 文件系统修改内核行为的一个示例

写入 3 表示清理文件页、目录项、Inodes 等各种缓存



#### 场景1: 磁盘和文件写案例

在第一个终端，运行 vmstat

```shell
# 每隔1秒输出1组数据
$ vmstat 1
procs -----------memory--------- ---swap-- -----io---- -system-- ------cpu------
r   b swpd    free  buff   cache si     so bi       bo in     cs us sy  id wa st
0   0    0 7743608  1112   92168  0      0  0        0 52    152  0  1 100  0  0
0   0    0 7743608  1112   92168  0      0  0        0 36     92  0  0 100  0  0
```

内存部分的 buff 和 cache，以及 io 部分的 bi 和 bo 是需要关注的重点。

- buff 和 cache 就是前面看到的 Buffers 和 Cache，单位是 KB
- bi 和 bo 分别表示块设备读取和写入的大小，单位是块/秒，linux 中块的大小是 1KB，这个单位也就等价于 KB/s



在第二个终端执行 dd 命令，通过读取随机设备，生成一个 500MB 大小的文件

```shell
$ dd if=/dev/urandom of=/tmp/file bs=1M count=500
```



再回到第一个终端，观察 Buffer 和 Cache 的变化情况

```shell
procs ---------memory--------- ---swap-- -----io---- -system-- ------cpu------
r   b swpd    free buff  cache si     so  bi      bo in     cs us sy  id wa st
0   0    0 7499460 1344 230484 0       0   0       0 29    145  0  0 100  0  0
1   0    0 7338088 1752 390512 0       0 488       0 39    558  0 47  53  0  0
1   0    0 7158872 1752 568800 0       0   0       4 30    376  1 50  49  0  0
1   0    0 6980308 1752 747860 0       0   0       0 24    360  0 50  50  0  0
0   0    0 6977448 1752 752072 0       0   0       0 29    138  0  0 100  0  0
0   0    0 6977440 1760 752080 0       0   0     152 42    212  0  1  99  1  0
...
0   1    0 6977216 1768 752104 0       0   4  122880 33    234  0  1  51 49  0
0   1    0 6977440 1768 752108 0       0   0   10240 38    196  0  0  50 50  0
```

在 dd 命令运行时，Cache 在不停地增常，Buffer 基本保持不变

- 在 Cache 刚开始增长时，块设备 IO 很少，bi 只出现了一次 488KB/s，bo 则只有一次 4KB，过一段时间后才会出现大量的块设备写，比如 bo 变成了122880
- 当 dd 命令结束后，Cache 不再增长，但块设备写还会持续一段时间，并且多次 IO 写的结果加起来才是 dd 要写的 500M 的数据



#### 场景2: 磁盘和文件读案例

dd 读取文件数据

```shell
$ dd if=/tmp/file of=/dev/null
```



观察内存和 IO 变化

```shell
procs ---------memory--------- ---swap-- ---io--- -system- ------cpu-----
r   b swpd    free buff  cache si     so    bi bo in    cs us sy id wa st
0   1    0 7724164 2380 110844  0      0 16576  0 62   360  2  2 76 21  0
0   1    0 7691544 2380 143472  0      0 32640  0 46   439  1  3 50 46  0
0   1    0 7658736 2380 176204  0      0 32640  0 54   407  1  4 50 46  0
0   1    0 7626052 2380 208908  0      0 32640 40 44   422  2  2 50 46  0
```



### 小节

Buffer 是对磁盘数据的缓存，Cache 是对文件数据的缓存，它们既会用在读请求中，也会用在写请求中

- 从写的角度来说，不仅可以优化磁盘和文件的写入，对应用程序也有好处，应用程序可以在数据真正落盘前，就返回去做其他工作
- 从读的角度来说，既可以加速读取那些需要频繁访问的数据，也降低了频繁 IO 对磁盘的压力



## Memory 分析套路

**系统内存指标**，比如已用内存、剩余内存、共享内存、可用内存、缓存和缓冲区的用量等。

- 已用内存和剩余内存，就是已经使用和还未使用的内存
- 共享内存，是通过 tmpfs 实现的，它的大小也就是 tmpfs 使用的内存大小，tmpfs 其实也是一种特殊的缓存
- 可用内存，是新进程可以使用的最大内存，它包括剩余内存和可回收缓存
- 缓存，包括两部分，一部分是磁盘读取文件的页缓存，用来缓存从磁盘读取的数据，可以加快以后再次访问的速度；另一部分，则是 Slab 分配器中的可回收内存
- 缓冲区，是对原始磁盘块的临时存储，用来缓存将要写入磁盘的数据，内核就可以把分散的写集中起来统一优化磁盘写入



**进程内存指标**，比如进程的虚拟内存、常驻内存、共享内存以及 Swap 内存等。

常驻内存一般会换算成占系统总内存的百分比，也就是进程的内存使用率。

- 虚拟内存，包括进程代码段、数据段、共享内存、已经申请的堆内存和已经换出的内存等，已经申请的内存即使还没有分配物理内存也算作虚拟内存
- 常驻内存，是进程实际使用的物理内存，不包括 Swap 和共享内存
- 共享内存，既包括与其他进程共同使用的真实共享内存，还包括加载的动态链接库以及程序的代码段等
- Swap 内存，是指通过 Swap 换出到磁盘的内存，比如 Swap 的已用空间、剩余空间、换入速度、换出速度等
  - 已用空间和剩余空间，表示已经使用和没有使用的内存空间
  - 换入、换出速度，表示每秒钟换入和换出内存的大小
- 缺页异常，系统调用内存分配请求后并不会立刻为其分配物理内存，而是在请求首次访问时通过缺页异常来分配
  - 可以直接从物理内存中分配时，被称为次缺页异常
  - 需要磁盘 IO 介入（比如 Swap）时，被称为主缺页异常



### 根据指标找工具（内存性能）

| 内存指标                         | 性能工具                                     |
| -------------------------------- | -------------------------------------------- |
| 系统已用、可用、剩余内存         | free<br />vmstat<br />sar<br />/proc/meminfo |
| 进程虚拟内存、常驻内存、共享内存 | ps<br />top                                  |
| 进程内存分布                     | Pmap                                         |
| 进程 Swap 换出内存               | top<br />/proc/pid/status                    |
| 进程缺页异常                     | ps<br />top                                  |
| 系统换页情况                     | sar                                          |
| 缓存/缓冲区用量                  | free<br />vmstat<br />sar<br />cachestat     |
| 缓存/缓冲区命中率                | cachetop                                     |
| SWAP 已用空间和剩余空间          | free<br />sar                                |
| SWAP 换入换出                    | vmstat                                       |
| 内存泄漏检测                     | memleak<br />valgrind                        |
| 指定文件的缓存大小               | pcstat                                       |



### 根据工具查指标（内存性能）

| 性能工具                  | 内存指标                                                     |
| ------------------------- | ------------------------------------------------------------ |
| free<br />/proc/meminfo   | 系统已用、可用、剩余内存以及缓存和缓冲区的使用量             |
| top<br />ps               | 进程虚拟、常驻、共享内存以及缺页异常                         |
| vmstat                    | 系统剩余内存、缓存、缓冲区、换入、换出                       |
| sar                       | 系统内存换页情况、内存使用率、缓存和缓冲区用量以及 Swap 使用情况 |
| cachestat                 | 系统缓存和缓冲区的命中率                                     |
| cachetop                  | 进程缓存和缓冲区的命中率                                     |
| slabtop                   | 系统 Slab 缓存使用情况                                       |
| /proc/pid/status          | 进程 Swap 内存等                                             |
| /proc/pid/smaps<br />pmap | 进程地址空间和内存状态                                       |
| valgrind                  | 进程内存错误检查器，用来检测内存初始化、泄漏、越界访问等各种内存错粗 |
| memleak                   | 内存泄漏检测                                                 |
| pcstat                    | 查找指定文件的缓存情况                                       |



### 快速分析内存性能瓶颈

- 先用 free 和 top，查看系统整体的内存使用情况
- 再用 vmstat 和 pidstat，查看一段时间的趋势，从而判断出内存问题的类型
- 最后进行详细分析，比如内存分配分析、缓存/缓冲区分析、具体进程的内存使用分析等

![memory_analyse](img/memory_analyse.png)
