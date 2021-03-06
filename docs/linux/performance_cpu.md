# linux 性能优化

**CPU 篇**

https://time.geekbang.org/column/intro/100020901?tab=catalog



## 1. 平均负载

单位时间内，系统处于**可运行状态**和**不可中断状态**的平均进程数，也就是平均活跃进程数，它和 CPU 使用率并没有直接关系。

- 可运行状态的进程，是指正在使用 CPU 或者正在等待 CPU 的进程，也就是我们常用 ps 命令看到的，处于 R 状态（Running 或 Runnable）的进程。
- 不可中断状态的进程，是正处于内核态关键流程中的进程，并且这些流程是不可打断的，比如最常见的是等待硬件设备的 I/O 响应，也就是我们在 ps 命令中看到的 D 状态（Uninterruptible Sleep，也称为 Disk Sleep）的进程。不可中断状态实际上是系统对进程和硬件设备的一种保护机制。



**平均负载最理想的情况是等于 CPU 个数**

```shell
grep 'model name' /proc/cpuinfo | wc -l

lscpu | grep 'CPU(s):'
```

**当平均负载高于 CPU 数量 70% 的时候**，就应该分析排查负载高的问题了。一旦负载过高，就可能导致进程响应变慢，进而影响服务的正常功能。但 70% 这个数字并不是绝对的，最推荐的方法，还是把系统的平均负载监控起来，然后根据更多的历史数据，判断负载的变化趋势。**当发现负载有明显升高趋势时，比如说负载翻倍了，再去做分析和调查**。



### 案例

- stress 是一个 Linux 系统压力测试工具，用作异常进程模拟平均负载升高的场景。

- mpstat 是一个常用的多核 CPU 性能分析工具，用来实时查看每个 CPU 的性能指标，以及所有 CPU 的平均指标。
- pidstat 是一个常用的进程性能分析工具，用来实时查看进程的 CPU、内存、I/O 以及上下文切换等性能指标。



#### 场景一：CPU密集型进程

安装 stress、sysstat

```shell
apt install stress sysstat
```



stress 命令模拟一个 CPU 使用率 100% 的场景

```shell
$ stress -c 1 -t 600
```



运行 uptime 查看平均负载的变化情况

```shell
# -d 参数表示高亮显示变化的区域
$ watch -d uptime
..., load average: 1.00, 0.75, 0.39
```



mpstat 查看 CPU 使用率的变化情况

```shell
# -P ALL 表示监控所有CPU，后面数字5表示间隔5秒后输出一组数据
$ mpstat -P ALL 5
Linux 4.15.0 (ubuntu) 09/22/18 _x86_64_ (2 CPU)
13:30:06 CPU   %usr %nice %sys %iowait %irq %soft %steal %guest %gnice  %idle
13:30:11 all  50.05  0.00 0.00    0.00 0.00  0.00   0.00   0.00   0.00  49.95
13:30:11   0   0.00  0.00 0.00    0.00 0.00  0.00   0.00   0.00   0.00 100.00
13:30:11   1 100.00  0.00 0.00    0.00 0.00  0.00   0.00   0.00   0.00   0.00
```



pid 查看消耗 CPU 的进程

```shell
# 间隔5秒后输出一组数据
$ pidstat -u 5 1
13:37:07 UID  PID   %usr %system %guest %wait   %CPU CPU Command
13:37:12   0 2962 100.00    0.00   0.00  0.00 100.00   1 stress
```

stress 进程的 CPU 使用率为 100%。



#### 场景二：IO密集型进程

stress 模拟IO压力，不停地 sync

```shell
$ stress -i 1 -t 600
```



uptime 查看平均负载的变化情况

```shell
$ watch -d uptime
..., load average: 1.06, 0.58, 0.37
```



mpstat 查看 CPU 使用率的变化情况

```shell
# 显示所有CPU的指标，并在间隔5秒输出一组数据
$ mpstat -P ALL 5 1Linux 4.15.0 (ubuntu) 09/22/18 _x86_64_ (2 CPU)
13:41:28 CPU %usr %nice  %sys %iowait %irq %soft %steal %guest %gnice %idle
13:41:33 all 0.21  0.00 12.07   32.67 0.00  0.21   0.00   0.00   0.00 54.84
13:41:33   0 0.43  0.00 23.87   67.53 0.00  0.43   0.00   0.00   0.00  7.74
13:41:33   1 0.00  0.00  0.81    0.20 0.00  0.00   0.00   0.00   0.00 98.99
```

1 分钟的平均负载会慢慢增加到 1.06，其中一个 CPU 的系统 CPU 使用率升高到了 23.87，而 iowait 高达 67.53%。

这说明，平均负载的升高是由于 iowait 的升高。



pidstat 查看IO消耗高的进程

```shell
# 间隔5秒后输出一组数据，-u表示CPU指标
$ pidstat -u 5 1
Linux 4.15.0 (ubuntu) 09/22/18 _x86_64_ (2 CPU)
13:42:08 UID  PID %usr %system %guest %wait  %CPU CPU Command
13:42:13   0  104 0.00    3.39   0.00  0.00  3.39   1 kworker/1:1H
13:42:13   0  109 0.00    0.40   0.00  0.00  0.40   0 kworker/0:1H
13:42:13   0 2997 2.00   35.53   0.00  3.99 37.52   1 stress
13:42:13   0 3057 0.00    0.40   0.00  0.00  0.40   0 pidstat
```



#### 场景三：大量进程

stress 模拟大量进程

```shell
$ stress -c 8 -t 600
```



uptime 查看平均负载的变化情况

```shell
$ uptime
..., load average: 7.97, 5.93, 3.02
```



pid 查看进程状态

```shell
# 间隔5秒后输出一组数据
$ pidstat -u 5 1
14:23:25 UID  PID  %usr %system %guest %wait  %CPU CPU Command
14:23:30   0 3190 25.00    0.00   0.00 74.80 25.00   0 stress
14:23:30   0 3191 25.00    0.00   0.00 75.20 25.00   0 stress
14:23:30   0 3192 25.00    0.00   0.00 74.80 25.00   1 stress
14:23:30   0 3193 25.00    0.00   0.00 75.00 25.00   1 stress
14:23:30   0 3194 24.80    0.00   0.00 74.60 24.80   0 stress
14:23:30   0 3195 24.80    0.00   0.00 75.00 24.80   0 stress
14:23:30   0 3196 24.80    0.00   0.00 74.60 24.80   1 stress
14:23:30   0 3197 24.80    0.00   0.00 74.80 24.80   1 stress
14:23:30   0 3200  0.00    0.20   0.00  0.20  0.20   0 pidstat
```

8 个进程在争抢 2 个 CPU，每个进程等待 CPU 的时间（也就是代码块中的 %wait 列）高达 75%。这些超出 CPU 计算能力的进程，最终导致 CPU 过载。



### 小结

- 平均负载高有可能是 CPU 密集型进程导致的
- 平均负载高并不一定代表 CPU 使用率高，还有可能是 I/O 更繁忙了
- 当发现负载高的时候，你可以使用 mpstat、pidstat 等工具，辅助分析负载的来源



## 2. 上下文切换

CPU 寄存器，是 CPU 内置的容量小、速度极快的内存。程序计数器，是用来存储 CPU 正在执行的指令位置、或者即将执行的下一条指令位置。它们都是 CPU 在运行任何任务前，必须的依赖环境，叫做 CPU 上下文。

CPU 上下文切换，就是先把前一个任务的 CPU 上下文（也就是 CPU 寄存器和程序计数器）保存起来，然后加载新任务的上下文到这些寄存器和程序计数器，最后再跳转到程序计数器所指的新位置，运行新任务。

CPU 上下文切换场景分为：进程上下文切换、线程上下文切换、中断上下文切换。



如何查看系统的上下文切换情况？

1. vmstat

vmstat 是一个常用的系统性能分析工具，主要用来分析系统的内存使用情况，也常用来分析 CPU 上下文切换和中断的次数。

```shell
# 每隔5秒输出1组数据
$ vmstat 5
procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu----- 
r   b swpd   free   buff    cache si     so bi       bo in     cs us sy id wa st 
0   0   0 7005360  91564   818900  0      0  0        0 25     33  0  0 100 0  0
```

- cs (context switch) 是每秒上下文切换的次数
- in (interruput) 是每秒中断的次数
- r (Running or Runnable) 是就绪队列的长度，也就是正在运行和等待 CPU 的进程数
- b (Blocked) 是处于中断睡眠状态的进程数



2. pidstat

vmstat 只给出了系统总体的上下文切换情况，要想查看每个进程的详细情况，就需要使用 pidstat，加上 -w 选项，就可以查看每个进程上下文切换的情况。

```shell
# 每隔5秒输出1组数据
$ pidstat -w 5
Linux 4.15.0 (ubuntu) 09/23/18 _x86_64_ (2 CPU)

08:18:26 UID PID cswch/s nvcswch/s Command
08:18:31   0   1    0.20      0.00 systemd
08:18:31   0   8    5.40      0.00 rcu_sched
...
```

- cswch 表示每秒自愿上下文切换的次数。
  - 指进程无法获取所需资源导致的上下文切换
  - 比如 IO、内存等系统资源不足时就会发生资源上下文切换
- nvcswch 表示每秒非自愿上下文切换的次数。
  - 指进程由于时间片已到等原因，被系统强制调度，进而发生的上下文切换
  - 比如大量进程都在争抢 CPU 时，就容易发生非自愿上下文切换



### 案例

sysbench 模拟系统多线程调度的瓶颈

```shell
# 以10个线程运行5分钟的基准测试，模拟多线程切换的问题
$ sysbench --threads=10 --max-time=300 threads run
```



接着第二个终端运行 vmstat，观察上下文切换情况

```shell
# 每隔1秒输出1组数据（需要Ctrl+C才结束）
$ vmstat 1
procs -----------memory--------- ---swap-- -----io---- ----system--- ------cpu-----
r   b swpd   free   buff   cache si     so bi       bo    in      cs us sy id wa st
6   0   0 6487428 118240 1292772  0      0  0        0  9019 1398830 16 84  0  0  0
8   0   0 6487428 118240 1292772  0      0  0        0 10191 1392312 16 84  0  0  0
```

cs 列的上下文切换次数从 35 骤然上升到 139 万，同时注意其他几个指标：

- r：就绪队列的长度已经到达 8，远超过了系统 CPU 的个数 2，肯定有大量的 CPU 竞争
- us、sy：这两列的 CPU 使用率加起来上升到 100%，其中系统 CPU 使用率，也就是 sy 高达 84%，说明 CPU 主要是被内核占用了
- in：中断次数也上升到 1万 左右，说明中断处理也是潜在问题

综合几个指标，说明，系统的就绪队列过长，也就是正在运行和等待 CPU 的进程数过多，导致了大量的上下文切换，而上下文切换又导致了系统 CPU 的占用率升高。



pidstat 分析导致问题的进程

```shell
# 每隔1秒输出1组数据（需要 Ctrl+C 才结束）
# -w参数表示输出进程切换指标，而-u参数则表示输出CPU使用指标
$ pidstat -w -u 1
08:06:33 UID   PID  %usr %system %guest %wait   %CPU CPU Command
08:06:34   0 10488 30.00  100.00   0.00  0.00 100.00   0 sysbench
08:06:34   0 26326  0.00    1.00   0.00  0.00   1.00   0 kworker/u4:2

08:06:33  UID   PID cswch/s nvcswch/s Command
08:06:34    0     8   11.00      0.00 rcu_sched
08:06:34    0    16    1.00      0.00 ksoftirqd/1
08:06:34    0   471    1.00      0.00 hv_balloon
08:06:34    0  1230    1.00      0.00 iscsid
08:06:34    0  4089    1.00      0.00 kworker/1:5
08:06:34    0  4333    1.00      0.00 kworker/0:3
08:06:34    0 10499    1.00    224.00 pidstat
08:06:34    0 26326  236.00      0.00 kworker/u4:2
08:06:34 1000 26784  223.00      0.00 sshd
```

pidstat 的输出可以发现，CPU 使用率的升高时 sysbench 导致的，它的 CPU 使用率已经达到了 100%。但上下文切换则是来自其他进程，包括非自愿上下文切换频率最高的 pidstat，以及自愿上下文切换频率最高的内核线程 kworker 和 sshd。



pidstat -t 查看线程指标

```shell
# 每隔1秒输出一组数据（需要 Ctrl+C 才结束）
# -wt 参数表示输出线程的上下文切换指标
$ pidstat -wt 1
08:14:05 UID  TGID   TID  cswch/s nvcswch/s Command
...
08:14:05   0 10551     -     6.00      0.00 sysbench
08:14:05   0     - 10551     6.00      0.00 |__sysbench
08:14:05   0     - 10552 18911.00 103740.00 |__sysbench
08:14:05   0     - 10553 18915.00 100955.00 |__sysbench
08:14:05   0     - 10554 18827.00 103954.00 |__sysbench
...
```

虽然 sysbench 进程的上下文切换次数看起来并不多，但是它的子线程的上下文切换次数却很多。



如何才能知道中断发生的类型呢？

可以从 /proc/interrupts 中读取

```shell
# -d 参数表示高亮显示变化的区域
$ watch -d cat /proc/interrupts
        CPU0    CPU1
...
RES: 2450431 5279697 Rescheduling interrupts
...
```

变化速度最快的是中调度中断（RES），这个中断类型表示唤醒空闲状态的 CPU 来调度新的任务运行。这是多处理器系统（SMP）中，调度器用来分散任务到不同 CPU 的机制，通常也被称为处理器间中断。



### 小结

- 自愿上下文切换变多了，说明进程都在等待自愿，有可能发生了 IO 等其他问题
- 非自愿上下文切换变多了，说明进程都在被强制调度，也就是都在争抢 CPU，说明 CPU 的确成了瓶颈
- 中断次数变多了，说明 CPU 被中断处理程序占用，还需要通过查看 /proc/interrupts 来分析具体的中断类型



## 3. CPU 使用率

为了维护 CPU 时间，linux 通过事先定义的节拍率（内核中表示为 Hz），触发时间中断，并使用全局变量 Jiffies 记录了开机以来的节拍数。每发生一次时间中断，Jiffies 的值就加 1。

节拍率 Hz 是内核的可配选项，可以设置为 100、250、1000 等。不同的系统可能设置不同数值，可以通过查询 /boot/config 内核选项来查看它的配置值。

```shell
$ grep 'CONFIG_HZ=' /boot/config-$(uname -r)
CONFIG_HZ=250
```

为了方便用户空间程序，内核还提供了一个用户空间节拍率 USER_HZ，它总是固定为 100，也就是 1/100 秒。用户空间程序并不需要关心内核中 Hz 被设置成了多少，因为它看到的总是固定值 USER_HZ。

linux 通过 /proc 虚拟文件系统，向用户空间提供了系统内部状态的信息，而 /proc/stat 提供的就是系统的 CPU 和任务统计信息。

```shell
# 只保留各个CPU的数据
$ cat /proc/stat | grep ^cpu
cpu  280580 7407 286084 172900810 83602 0 583 0 0 0
cpu0 144745 4181 176701  86423902 52076 0 301 0 0 0
cpu1 135834 3226 109383  86476907 31525 0 282 0 0 0
```

每列数值表示不同场景下 CPU 的累加节拍数，它的单位是 USER_HZ，也就是 10ms（1/100 秒），就是不同场景下的 CPU 时间。

- user（缩写为 us），代表用户态 CPU 时间。它不包括下面的 nice 时间，但包括了 guest 时间。
- nice（缩写为 ni），代表低优先级用户态 CPU 时间，也就是进程的 nice 值被调整为 1-19 之间时的 CPU 时间。nice 可取值范围是 -20 到 19，数值越大，优先级反而越低。
- system（缩写为 sys），代表内核态 CPU 时间。
- idle（缩写为 id），代表空闲时间。它不包括等待 IO 的时间（iowait）。
- iowait（缩写为 wa），代表等待 IO 的 CPU 时间。
- irq（缩写为 hi），代表处理硬中断的 CPU 时间。
- softirq（缩写为 si），代表处理软中断的 CPU 时间。
- steal（缩写为 st），代表当系统运行在虚拟机中的时候，被其他虚拟机占用的 CPU 时间。
- guest（缩写为 guest），代表通过虚拟化运行其他操作系统的时间，也就是运行虚拟机的 CPU 时间。
- guest_nice（缩写为 gnice），代表以低优先级运行虚拟机的时间。

CPU 使用率，就是除了空闲时间外的其他时间占总 CPU 时间的百分比。

性能分析工具给出的都是间隔一段时间的平均 CPU 使用率，所以要注意间隔时间的设置，特别是用多个工具对比分析时，一定要保证用的是相同的间隔时间。



查看 CPU 使用率

- top，显示了系统总体的 CPU 和内存使用情况，以及各个进程的资源使用情况
- ps，只显示了每个进程的资源使用情况

```shell
# 默认每3秒刷新一次
$ top
top - 11:58:59 up 9 days, 22:47, 1 user, load average: 0.03, 0.02, 0.00
Tasks: 123 total, 1 running, 72 sleeping, 0 stopped, 0 zombie
%Cpu(s): 0.3 us, 0.3 sy, 0.0 ni, 99.3 id, 0.0 wa, 0.0 hi, 0.0 si, 0.0 st
KiB Mem : 8169348 total, 5606884 free, 334640 used, 2227824 buff/cache
KiB Swap:       0 total,       0 free,      0 used. 7497908 avail Mem

PID USER PR  NI  VIRT  RES  SHR S %CPU %MEM   TIME+ COMMAND
  1 root 20   0 78088 9288 6696 S  0.0  0.1 0:16.83 systemd
  2 root 20   0     0    0    0 S  0.0  0.0 0:00.05 kthreadd
  4 root  0 -20     0    0    0 I  0.0  0.0 0:00.00 kworker/0:0H
...
```



pidstat，可以查看每个进程 CPU 使用情况

- 用户态 CPU 使用率（%usr）
- 内核态 CPU 使用率（%system）
- 运行虚拟机 CPU 使用率（%guest）
- 等待 CPU 使用率（%wait）
- 总的 CPU 使用率（%CPU）

```shell
# 每隔1秒输出一组数据，共输出5组
$ pidstat 1 5
15:56:02 UID   PID %usr %system %guest %wait %CPU CPU Command
15:56:03   0 15006 0.00    0.99   0.00  0.00 0.99   1 dockerd

...

Average: UID   PID %usr %system %guest %wait %CPU CPU Command
Average:   0 15006 0.00    0.99   0.00  0.00 0.99   - dockerd
```



CPU 使用率过高分析：使用 perf 分析 CPU 性能问题

1. perf top，类似于 top，能够实时显示占用 CPU 时钟最多的函数或者指令，可以用来查找热点函数

```shell
$ perf top
Samples: 833 of event 'cpu-clock', Event count (approx.): 97742399
Overhead Shared Object Symbol
   7.28%          perf [.] 0x00000000001f78a4
   4.72%      [kernel] [k] vsnprintf
   4.32%      [kernel] [k] module_get_kallsym
   3.65%      [kernel] [k] _raw_spin_unlock_irqrestore
...
```

第一行包括三个数据，分别是采样数、事件类型和事件总数量。

- Overhead，是该符号的性能事件在所有采样中的比例，用百分比来表示
- Shared，是该函数或者指令所在的动态共享对象，如内核、进程名、动态链接库名、内核模块名等
- Object，是动态共享对象的类型。[.]表示用户空间的可执行程序、或者动态链接库，[k]表示内核空间
- Symbol，是符号名，也就是函数名。当函数名未知时，用十六进制的地址来表示。

2. perf record、perf report

perf top 虽然实时展示了系统的性能信息，但它的缺点时并不保存数据，也就无法用于离线或者后续的分析。perf record 则提供了保存数据的功能，保存后的数据需要用 perf report 解析展示。

```shell
$ perf record # 按Ctrl+C终止采样
[ perf record: Woken up 1 times to write data ]
[ perf record: Captured and wrote 0.452 MB perf.data (6093 samples) ]

$ perf report # 展示类似于perf top的报告
```

在实际使用中，还可以加上 -g 参数，开启调用关系的采样，方便根据调用链来分析性能问题。



### 案例

#### 场景一

在第一个终端执行命令来运行 Nginx、PHP

```shell
$ docker run --name nginx -p 10000:80 -itd feisky/nginx
$ docker run --name phpfpm -itd --network container:nginx feisky/php-fpm
```



在第二个终端使用 curl 访问 Nginx

```shell
# 192.168.0.10是第一台虚拟机的IP地址
$ curl http://192.168.0.10:10000/
It works!
```



在第二个终端运行 ab 命令

```shell
# 并发10个请求测试Nginx性能，总共测试100个请求
$ ab -c 10 -n 100 http://192.168.0.10:10000/
This is ApacheBench, Version 2.3 <$Revision: 1706008 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd,
...
Requests per second: 11.63 [#/sec] (mean)
Time per request: 859.942 [ms] (mean)
...
```



继续在第二个终端运行 ab 命令

```shell
$ ab -c 10 -n 10000 http://192.168.0.10:10000/
```



回到第一个终端运行 top，并按下 1，切换到每个 CPU 的使用率

```shell
$ top
...
%Cpu0 : 98.7 us, 1.3 sy, 0.0 ni, 0.0 id, 0.0 wa, 0.0 hi, 0.0 si, 0.0 st
%Cpu1 : 99.3 us, 0.7 sy, 0.0 ni, 0.0 id, 0.0 wa, 0.0 hi, 0.0 si, 0.0 st
...
  PID   USER PR NI   VIRT   RES  SHR S %CPU %MEM   TIME+ COMMAND
21514 daemon 20  0 336696 16384 8712 R 41.9  0.2 0:06.00 php-fpm
21513 daemon 20  0 336696 13244 5572 R 40.2  0.2 0:06.08 php-fpm
21515 daemon 20  0 336696 16384 8712 R 40.2  0.2 0:05.67 php-fpm
21512 daemon 20  0 336696 13244 5572 R 39.9  0.2 0:05.87 php-fpm
21516 daemon 20  0 336696 16384 8712 R 35.9  0.2 0:05.61 php-fpm
```

可以确认，正是 php-fpm 进程导致 CPU 使用率升高。



如何知道是 php-fpm 的哪个函数导致 CPU 使用率升高呢？

在第一个终端运行 perf 命令

```shell
# -g开启调用关系分析，-p指定php-fpm的进程号21515
$ perf top -g -p 21515
```

按方向键切换到 php-fpm，再按下回车键展开调用关系，发现调用关系最终到了 sqrt 和 add_function，分析定位问题。



#### 场景二

在第一个终端，运行 Nginx 和 PHP

```shell
docker run --name nginx -p 10000:80 -itd feisky/nginx:sp$ docker run --name phpfpm -itd --network container:nginx feisky/php-fpm:sp
```



在第二个终端，持用 curl 访问，确认 Nginx 已正常启动

```shell
# 192.168.0.10是第一台虚拟机的IP地址
$ curl http://192.168.0.10:10000/
It works!
```



在第二个终端运行 ab 进行性能测试

```shell
# 并发100个请求测试Nginx性能，总共测试1000个请求
$ ab -c 100 -n 1000 http://192.168.0.10:10000/
This is ApacheBench, Version 2.3 <$Revision: 1706008 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd,
...
Requests per second: 87.86 [#/sec] (mean)
Time per request: 1138.229 [ms] (mean)
...
```



继续 ab 压测

```shell
ab -c 5 -t 600 http://192.168.0.10:10000/
```



在第一个终端运行 top 查看系统 CPU 使用情况

```shell
$ top
...
%Cpu(s): 80.8 us, 15.1 sy, 0.0 ni, 2.8 id, 0.0 wa, 0.0 hi, 1.3 si, 0.0 st
...
  PID     USER PR NI    VIRT   RES   SHR S %CPU %MEM   TIME+ COMMAND
 6882     root 20  0    8456  5052  3884 S  2.7  0.1 0:04.78 docker-containe
 6947 systemd+ 20  0   33104  3716  2340 S  2.7  0.0 0:04.92 nginx
 7494   daemon 20  0  336696 15012  7332 S  2.0  0.2 0:03.55 php-fpm
 7495   daemon 20  0  336696 15160  7480 S  2.0  0.2 0:03.55 php-fpm
10547   daemon 20  0  336696 16200  8520 S  2.0  0.2 0:03.13 php-fpm
10155   daemon 20  0  336696 16200  8520 S  1.7  0.2 0:03.12 php-fpm
10552   daemon 20  0  336696 16200  8520 S  1.7  0.2 0:03.12 php-fpm
15006     root 20  0 1168608 66264 37536 S  1.0  0.8 9:39.51 dockerd
 4323     root 20  0       0     0     0 I  0.3  0.0 0:00.87 kworker/u4:1
...
```

系统 CPU 使用率（%Cpu）比较高：用户 CPU 使用率达到了 80%，系统 CPU 为 15.1%，空闲 CPU 只有 2.8%。

用户 CPU 使用率高，分析一下进程列表，没有发现高 CPU 使用率的进程。



pid 分析进程 CPU 使用情况

```shell
# 间隔1秒输出一组数据（按Ctrl+C结束
$ pidstat 1
...
04:36:24 UID   PID %usr %system %guest %wait %CPU CPU Command
04:36:25   0  6882 1.00    3.00   0.00  0.00 4.00   0 docker-containe
04:36:25 101  6947 1.00    2.00   0.00  1.00 3.00   1 nginx
04:36:25   1 14834 1.00    1.00   0.00  1.00 2.00   0 php-fpm
04:36:25   1 14835 1.00    1.00   0.00  1.00 2.00   0 php-fpm
04:36:25   1 14845 0.00    2.00   0.00  2.00 2.00   1 php-fpm
04:36:25   1 14855 0.00    1.00   0.00  1.00 1.00   1 php-fpm
04:36:25   1 14857 1.00    2.00   0.00  1.00 3.00   0 php-fpm
04:36:25   0 15006 0.00    1.00   0.00  0.00 1.00   0 dockerd
04:36:25   0 15801 0.00    1.00   0.00  0.00 1.00   1 pidstat
04:36:25   1 17084 1.00    0.00   0.00  2.00 1.00   0 stress
04:36:25   0 31116 0.00    1.00   0.00  0.00 1.00   0 atopacctd
...
```

所有进程的 CPU 使用率也都不高



重新分析 top 的输出

从头开始看 top 的输出，就绪队列中有 6 个 Running 状态的进程（6 running）

再查看进程列表，主要看 Running（R）状态的进程

发现 Nginx 和 php-fpm 都处于 Sleep（S）状态，而真正处于 Running（R）状态的，却是几个 stress 进程



使用 pidstat 分析 stress 进程，并且使用 -p 选项指定进程的 PID

```shell
$ pidstat -p 24344

16:14:55 UID PID %usr %system %guest %wait %CPU CPU Command
```

没有输出，ps 确认下进程

```shell
# 从所有进程中查找PID是24344的进程
$ ps aux | grep 24344
root 9628 0.0 0.0 14856 1096 pts/0 S+ 16:15 0:00 grep --color=auto 24344
```



再回到 top 发现刚才的 stress 进程不存在了，这次 stress 进程的 PID 不同了

- 第一个原因，进程在不停地崩溃重启，比如因为段错误、配置错误等等，这时进程在退出后可能又被监控系统自动重启了
- 第二个原因，这些进程都是短时进程，也就是在其他进程内部通过 exec 调用的外面命令，这些命令一般都只运行很短的时间就会结束，很难用 top 这种间隔时间比较长的工具发现

```shell
$ pstree | grep stress
         |-docker-containe-+-php-fpm-+-php-fpm---sh---stress
         |     |-3*[php-fpm---sh---stress---stress]
```



execsnoop 是一个专为短时进程设计的工具

```shell
# 按 Ctrl+C 结束
$ execsnoop
 PCOMM   PID  PPID RET ARGS
    sh 30394 30393   0
stress 30396 30394   0 /usr/local/bin/stress -t 1 -d 1
    sh 30398 30393   0
stress 30399 30398   0 /usr/local/bin/stress -t 1 -d 1
    sh 30402 30400   0
stress 30403 30402   0 /usr/local/bin/stress -t 1 -d 1
    sh 30405 30393   0
stress 30407 30405   0 /usr/local/bin/stress -t 1 -d 1
...
```



## 4. 不可中断进程

进程的状态

- R（Running、Runnable），表示进程在 CPU 的就绪队列中，正在运行或者在等待运行
- D（Disk Sleep），不可中断状态睡眠，一般表示进程正在跟硬件交互，并且交互过程不允许被其他进程或中断打断
- Z（Zombie），僵尸进程，表示进程实际上已经结束了但是父进程还没有回收它的资源（比如进程的描述符、PID）
- S（Interruptible Sleep），可中断状态睡眠，表示进程因为等待某个事件而被系统挂起，当进程等待的事件发生时，会被唤醒并进入 R 状态
- I（Idle），空闲状态，用在不可中断睡眠的内核线程上
- T（Stopped、Traced），表示进程处于暂停或者跟踪状态
  - 向一个进程发送 SIGSTOP 信号就会变成暂停状态（Stopped）
  - 向进程发送 SIGCONT 信号，进程又会恢复运行
  - 当用调试器（gdb）调试一个进程时，在使用断点中断进程后，进程就会变成跟踪状态
- X（Dead），表示进程已经消亡



### 案例

dstat 可以同时观察系统的 CPU、磁盘 IO、网络以及内存使用情况。



运行案例

```shell
$ docker run --privileged --name=app -itd feisky/app:iowait
```



ps 查看状态

```shell
$ ps aux | grep /app
root 4009 0.0 0.0  4376  1008 pts/0 Ss+ 05:51 0:00 /app
root 4287 0.6 0.4 37280 33660 pts/0  D+ 05:54 0:00 /app
root 4288 0.6 0.4 37280 33668 pts/0  D+ 05:54 0:00 /app
```

多个 app 进程已经启动，并且它们的状态分别是 Ss+ 和 D+。

- "S" 表示可中断睡眠状态
- "D" 表示不可中断睡眠状态
- "s" 表示这个进程是一个会话的领导进程
- "+" 表示前台进程组
  - 进程组表示一组相互关联的进程，比如每个子进程都是父进程所在组的成员
  - 会话指共享同一个控制终端的一个或多个进程组



top 分析系统的资源使用情况

```shell
# 按下数字 1 切换到所有 CPU 的使用情况，观察一会儿按 Ctrl+C 结束
$ top
top - 05:56:23 up 17 days, 16:45, 2 users, load average: 2.00, 1.68, 1.39
Tasks: 247 total, 1 running, 79 sleeping, 0 stopped, 115 zombie
%Cpu0 : 0.0 us, 0.7 sy, 0.0 ni, 38.9 id, 60.5 wa, 0.0 hi, 0.0 si, 0.0 st
%Cpu1 : 0.0 us, 0.7 sy, 0.0 ni, 4.7 id, 94.6 wa, 0.0 hi, 0.0 si, 0.0 st
...
 PID USER PR NI   VIRT   RES  SHR S %CPU %MEM   TIME+ COMMAND
4340 root 20  0  44676  4048 3432 R  0.3  0.0 0:00.05 top
4345 root 20  0  37280 33624  860 D  0.3  0.0 0:00.01 app
4344 root 20  0  37280 33624  860 D  0.3  0.4 0:00.01 app
   1 root 20  0 160072  9416 6752 S  0.0  0.1 0:38.59 systemd
...
```

可以问题：

- 第一行的平均负载（Load Average），过去 1 分钟、5 分钟和 15 分钟内的平均负载在依次减小，说明平均负载正在升高；而 1 分钟内的平均负载已经达到系统的 CPU 个数，说明系统很可能已经有了性能瓶颈
- 第二行的 Tasks，有 1 个正在运行的进程，但僵尸进程比较多，而且还在不停增加，说明有子进程在退出时没被清理
- 两个 CPU 的使用率，用户 CPU 和系统 CPU 都不高，但 iowait 分别时 60.5% 和 94.6%
- 每个进程的情况，CPU 使用率最高的进程只有 0.3%，但有两个进程处于 D 状态，它们可能在等待 IO



明确两个问题：

- iowait 太高了，导致系统的平均负载升高，甚至达到了系统 CPU 的个数
- 僵尸进程在不断增加，说明有程序没能正确处理子进程的资源



iowait 分析

dstat 可以同时查看 CPU 和 IO

```shell
# 间隔1秒输出10组数据
$ dstat 1 10
You did not select any stats, using -cdngy by default.
--total-cpu-usage--   -dsk/total-   -net/total-   ---paging--   ---system--
usr sys idl wai stl |  read writ  |  recv send  |   in  out   |  int  csw
  0   0  96   4   0 | 1219k 408k  |     0    0  |    0    0   |   42  885
  0   0   2  98   0 |   34M    0  |  198B 790B  |    0    0   |   42  138
  0   0   0 100   0 |   34M    0  |   66B 342B  |    0    0   |   42  135
  0   0  84  16   0 | 5633k    0  |   66B 342B  |    0    0   |   52  177
  0   3  39  58   0 |   22M    0  |   66B 342B  |    0    0   |   43  144
  0   0   0 100   0 |   34M    0  |  200B 450B  |    0    0   |   46  147
  0   0   2  98   0 |   34M    0  |   66B 342B  |    0    0   |   45  134
  0   0   0 100   0 |   34M    0  |   66B 342B  |    0    0   |   39  131
  0   0  83  17   0 | 5633k    0  |   66B 342B  |    0    0   |   46  168
  0   3  39  59   0 |   22M    0  |   66B 342B  |    0    0   |   37  134
```

从 dstat 的输出可以看到，每当 iowait 升高（wait）时，磁盘的读请求（read）都会很大，说明 iowait 的升高跟磁盘的读请求有关，很可能是磁盘读导致的。



运行 top 命令，观察 D 状态的进程

```shell
# 观察一会儿按 Ctrl+C 结束
$ top
...
 PID USER PR NI  VIRT   RES  SHR S %CPU %MEM   TIME+ COMMAND
4340 root 20  0 44676  4048 3432 R  0.3  0.0 0:00.05 top
4345 root 20  0 37280 33624  860 D  0.3  0.0 0:00.01 app
4344 root 20  0 37280 33624  860 D  0.3  0.4 0:00.01 app
...
```



pidstat 加上 -d 参数查看进程的读写情况

```shell
# -d 展示 I/O 统计数据，-p 指定进程号，间隔 1 秒输出 3 组数据
$ pidstat -d -p 4344 1 3
06:38:50 UID  PID kB_rd/s kB_wr/s kB_ccwr/s iodelay Command
06:38:51   0 4344    0.00    0.00      0.00       0 app
06:38:52   0 4344    0.00    0.00      0.00       0 app
06:38:53   0 4344    0.00    0.00      0.00       0 app
```

- kB_rd 表示每秒读的 KB 数
- kB_wr 表示每秒写的 KB 数
- iodelay 表示 IO 的延迟（单位是时钟周期），它们都是 0 表示此事没有任何的读写，说明问题不是 4344 进程导致的



查看所有进程的读写情况

```shell
# 间隔 1 秒输出多组数据 (这里是 20 组)
$ pidstat -d 1 20
...
06:48:46 UID  PID  kB_rd/s kB_wr/s kB_ccwr/s iodelay Command
06:48:47   0 4615     0.00    0.00      0.00       1 kworker/u4:1
06:48:47   0 6080 32768.00    0.00      0.00     170 app
06:48:47   0 6081 32768.00    0.00      0.00     184 app

06:48:47 UID  PID kB_rd/s kB_wr/s kB_ccwr/s iodelay Command
06:48:48   0 6080    0.00    0.00      0.00     110 app

06:48:48 UID  PID kB_rd/s kB_wr/s kB_ccwr/s iodelay Command
06:48:49   0 6081    0.00    0.00      0.00     191 app

06:48:49 UID PID kB_rd/s kB_wr/s kB_ccwr/s iodelay Command

06:48:50 UID  PID  kB_rd/s kB_wr/s kB_ccwr/s iodelay Command
06:48:51   0 6082 32768.00    0.00      0.00       0 app
06:48:51   0 6083 32768.00    0.00      0.00       0 app

06:48:51 UID  PID  kB_rd/s kB_wr/s kB_ccwr/s iodelay Command
06:48:52   0 6082 32768.00    0.00      0.00     184 app
06:48:52   0 6083 32768.00    0.00      0.00     175 app

06:48:52 UID  PID kB_rd/s kB_wr/s kB_ccwr/s iodelay Command
06:48:53   0 6083    0.00    0.00      0.00     105 app
...
```

的确是 app 进程在进行磁盘读，并且每秒读的数据有 32MB。



strace 跟踪进程系统调用

```shell
$ strace -p 6082
strace: attach: ptrace(PTRACE_SEIZE, 6082): Operation not permitted
```



perf 追查

```shell
$ perf record -g
$ perf report
```

根据调用关系，发现 app 在通过系统调用 sys_read() 读取数据，并且从 new_sync_read 和 blkdev_direct_IO 看出进程正在对磁盘进行直接读，也就是绕过了系统缓存，每个读请求都会从磁盘直接读，这就可以解释 iowait 升高的原因。

直接读写磁盘，对 IO 敏感型应用（比如数据库系统）是很友好的，可以在应用中直接控制磁盘的读写。但在大部分情况下，最好还是通过系统缓存来优化磁盘 IO。



### 小结

进程状态包括：运行（R）、空闲（I）、不可中断睡眠（D）、可中断睡眠（S）、僵尸（Z）、暂停（T）

- 不可中断状态，表示进程正在跟硬件交互，为了保护进程数据和硬件的一致性，系统不允许其他进程或中断打断这个进程。进程长时间处于不可中断状态，通常表示系统有 IO 性能问题。
- 僵尸进程表示进程已经退出，但它的父进程还没有回收子进程占用的资源。短暂的僵尸状态通常不必理会，但长时间处于僵尸进程就应该注意了，可能有应用程序没有正常处理子进程的退出。



iowait 高不一定代表 IO 有性能瓶颈。当系统中只有 IO 类型的进程在运行时，iowait 也会很高，但实际上，磁盘的读写远没有达到性能瓶颈的程度。

碰到 iowait 升高，需要先用 dstat、pidstat 等工具，确认是不是磁盘 IO 的问题，然后再找是哪些进程导致了 IO。

等待 IO 的进程一般是不可中断状态，所以用 ps 命令找到 D 状态（不可中断状态）的进程，多为可以进程。



进程的不可中断状态是系统的一种保护机制，可以保证硬件的交互过程不被意外打断，短时间的不可中断状态是狠正常的。

但是，当进程长时间都处于不可中断状态时，就得小心0000

## 5. 僵尸进程

正常情况下，当一个进程创建了子进程后，它应该通过系统调用 wait 或者 waitpid 等待子进程结束，回收子进程的资源；而子进程在结束时，会向它的父进程发送 SIGCHLD 信号，所以父进程还可以注册 SIGCHLD 信号的处理函数，异步回收资源。

如果父进程没这么做，或是子进程执行太快，父进程还没来得及处理子进程状态，子进程就已经提前退出，这时的子进程就会变成僵尸进程。

僵尸进程持续的时间都比较短，在父进程回收它的资源后就会消亡；或者在父进程退出后，由 init 进程回收后也会消亡。

一旦父进程没有处理子进程的终止，还一直保持运行状态，那么子进程就会一直处于僵尸状态。大量的僵尸进程会用尽 PID 进程号，导致新进程不能创建。



僵尸进程是因为父进程没有回收子进程的资源而出现的，要解决掉它们就要找到它们的根，也就是找出父进程，然后在父进程里解决。

父进程最简单的就是运行 pstree

```shell
# a 表示输出命令行选项
# p 表PID
# s 表示指定进程的父进程
$ pstree -aps 3084
systemd,1
  └─dockerd,15006 -H fd://
    └─docker-containe,15024 --config /var/run/docker/containerd/containerd.toml
      └─docker-containe,3991 -namespace moby -workdir...
        └─app,4009
          └─(app,3084)
```



## 6. 软中断

linux 将中断处理过程分成两个阶段，也就是上半部和下半部：

- 上半部用来快速处理中断，它在中断禁止模式下运行，主要处理跟硬件紧密相关的或时间敏感的工作。上半部会打断 CPU 正在执行的任务，然后立即执行中断处理程序。
- 下半部用来延迟处理上半部未完成的工作，通常以内核线程的方式运行，每个 CPU 都对应一个软中断内核线程，名字为“ksoftirqd/CPU编号”。

软中断不只包括硬件设备中断处理程序的下半部，一些内核自定义的事件也属于软中断，比如内核调度和 RCU 锁（Read-Copy Update，RCU 是 linux 内核中最常用的锁之一）。



- /proc/softirqs 提供了软中断的运行情况
- /proc/interrupts 提供了硬中断的运行情况



```shell
$ cat /proc/softirqs
             CPU0    CPU1
      HI:       0       0
   TIMER:  811613 1972736
  NET_TX:      49       7
  NET_RX: 1136736 1506885
   BLOCK:       0       0
IRQ_POLL:       0       0
 TASKLET:  304787    3691
   SCHED:  689718 1897539
 HRTIMER:       0       0
     RCU: 1330771 1354737
```

- 注意软中断的类型，也就是第一列的内容。从第一列可以看到，软中断包括了 10 个类别，分别对应不同的工作类型。比如 NET_RX 表示网络接收中断，NET_TX 表示网络发送中断。
- 同一种软中断在不同 CPU 上的分布，也就是同一行的内容。正常情况下，同一种中断在不同 CPU 上的累积次数应该差不多。比如 NET_RX 在 CPU0 和 CPU1 上的中断次数基本是同一个数量级，相差不大。
- TASKLET 在不同 CPU 上的分布并不均匀。TASKLET 是最常用的软中断实现机制，每个 TASKLET 只运行一次就会结束，并且只在调用它的函数所在的 CPU 上运行。



可以用 ps 查看软中断内核线程的运行状况

```shell
$ ps aux | grep softirq
root  7 0.0 0.0 0 0 ? S Oct10 0:01 [ksoftirqd/0]
root 16 0.0 0.0 0 0 ? S Oct10 0:01 [ksoftirqd/1]
```

ps 的输出中，名字括在中括号里的，一般都是内核线程。



当软中断事件的频率过高时，内核线程也会因为 CPU 使用率过高而导致软中断处理不及时，进而引发网络收发延迟、调度缓慢等性能问题。



### 案例

- sar 是一个系统活动报告工具，既可以实时查看系统的当前活动，又可以配置保存和报告历史统计数据
- hping3 是一个可以构造 TCP/IP 协议数据包的工具，可以对系统进行安全审计、防火墙测试等
- tcpdump 是一个常用的网络抓包工具，常用来分析各种网络问题



在第一个终端，启动 Nginx

```shell
# 运行Nginx服务并对外开放80端口
$ docker run -itd --name=nginx -p 80:80 nginx
```



在第二个终端，使用 curl 访问 Nginx 监听的端口，确认 Nginx 正常启动

```shell
$ curl http://192.168.0.30/
<!DOCTYPE html>
<html>
<head>
<title>Welcoome to nginx!</title>
...
```

接着，运行 hping3 模拟 Nginx 客户端请求

```shell
# -S参数表示设置TCP协议的SYN（同步序列号），-p表示目的端口为80
# -i u100表示每隔100微秒发送一个网络帧
# 注：如果你在实践过程中现象不明显，可以尝试把100调小，比如调成10甚至1
$ hping3 -S -p 80 -i u100 192.168.0.30
```



在第一个终端运行 top，看一下系统整体的资源使用情况

```shell
# top运行后按数字1切换到显示所有CPU
$ top
top - 10:50:58 up 1 days, 22:10, 1 user, load average: 0.00, 0.00, 0.00
Tasks: 122 total, 1 running, 71 sleeping, 0 stopped, 0 zombie
%Cpu0 : 0.0 us, 0.0 sy, 0.0 ni, 96.7 id, 0.0 wa, 0.0 hi, 3.3 si, 0.0 st
%Cpu1 : 0.0 us, 0.0 sy, 0.0 ni, 95.6 id, 0.0 wa, 0.0 hi, 4.4 si, 0.0 st
...

 PID USER PR NI   VIRT   RES   SHR S %CPU %MEM   TIME+ COMMAND
   7 root 20  0      0     0     0 S  0.3  0.0 0:01.64 ksoftirqd/0
  16 root 20  0      0     0     0 S  0.3  0.0 0:01.97 ksoftirqd/1
2663 root 20  0 923480 28292 13996 S  0.3  0.3 4:58.66 docker-containe
3699 root 20  0      0     0     0 I  0.3  0.0 0:00.13 kworker/u4:0
3708 root 20  0  44572  4176  3512 R  0.3  0.1 0:00.07 top
   1 root 20  0 225384  9136  6724 S  0.0  0.1 0:23.25 systemd
   2 root 20  0      0     0     0 S  0.0  0.0 0:00.03 kthreadd
...
```

分析：

- 平均负载全是 0，就绪队列里面只有一个进程（1 running）
- 每个 CPU 的使用率都挺低，最高的 CPU1 的使用率也只有 4.4%，并不算高
- 进程列表，CPU 使用率最高的进程也只有 0.3%，还是不高

仔细看 top 的输出，两个 CPU 的使用率虽然分别只有 3.3% 和 4.4%，但都用在了软中断上；而从进程列表上可以看到，CPU 使用率最高的也是软中断进程 ksoftirqd，看起来软中断有点可疑。



查看软中断信息，可以用 watch 监听变化情况，-d 参数可以高亮变化的部分，从高亮部分看出哪些内容变化得更快。

```shell
$ watch -d cat /proc/softirqs
             CPU0    CPU1
      HI:       0       0
   TIMER: 1083906 2368646
  NET_TX:      53       9
  NET_RX: 1550643 1916776
   BLOCK:       0       0
IRQ_POLL:       0       0
 TASKLET:  333637    3930
   SCHED:  963675 2293171
 HRTIMER:       0       0
     RCU: 1542111 1590625
```

TIMER、NET_RX、SCHED、RCU 这几个软中断都在不停变化

其中 NET_RX，也就是网络数据包接收软中断的变化速率最快



sar 可以用来查看系统的网络收发情况，不仅可以观察网络收发的吞吐量（BPS，每秒收发的字节数），还可以观察网络收发的 PPS（每秒收发的网络帧数）。

在第一个终端中运行 sar，添加 -n DEV 参数显示网络收发的报告

```shell
# -n DEV 表示显示网络收发的报告，间隔1秒输出一组数据
$ sar -n DEV 1
15:03:46       IFACE  rxpck/s  txpck/s rxkB/s txkB/s rxcmp/s txcmp/s rxmcst/s %ifutil
15:03:47        eth0 12607.00  6304.00 664.86 358.11    0.00    0.00     0.00    0.01
15:03:47     docker0  6302.00 12604.00 270.79 664.66    0.00    0.00     0.00    0.00
15:03:47          lo     0.00     0.00   0.00   0.00    0.00    0.00     0.00    0.00
15:03:47 veth9f6bbcd  6302.00 12604.00 356.95 664.66    0.00    0.00     0.00    0.05
```

- 第一列：表示报告的时间
- 第二列：IFACE 表示网卡
- 第三、四列：rxpck/s 和 txpck/s 分别表示每秒接收、发送的网络帧数，也就是 PPS
- 第五、六列：rxkB/s 和 txkB/s 分别表示每秒接收、发送的千字节数，也就是 BPS
- 后面的其他参数基本接近0，可以先忽略

分析输出的内容可以发现：

- 对网卡 eth0 来说，每秒接收的网络帧数比较大，达到了 12607，而发送的网络帧数则比较小，只有 6304；每秒接收的千字节数只有 664 KB，而发送的千字节数更小，只有 358 KB
- docker0 和 veth9f6bbcd 的数据跟 eth0 基本一致，只是发送和接收相反，发送的数据较大而接收的数据较小，这是 linux 内部网桥转发导致的



在第一个终端运行 tcpdump 抓包

```shell
# -i eth0 只抓取eth0网卡，-n不解析协议名和主机名
# tcp port 80表示只抓取tcp协议并且端口号为80的网络帧
$ tcpdump -i eth0 -n tcp port 80
15:11:32.678966 IP 192.168.0.2.18238 > 192.168.0.30.80: Flags [S], seq 458303614, win 512, length 0
...
```

- 192.168.0.2.18238 > 192.168.0.30.80 ，表示网络帧从 192.168.0.2 的 18238 端口发送到 192.168.0.30 的 80 端口，也就是从运行 hping3 机器的 18238 端口发送网络帧，目的为 Nginx 所在机器的 80 端口
- Flags [S] 则表示这是一个 SYN 包

确认是从 192.168.0.2 发送过来的 SYNC FLOOD 攻击。



## CPU 分析套路

- CPU 使用率

  CPU 使用率描述了非空闲时间占总 CPU 时间的百分比，根据 CPU 上运行任务的不同，又被分为用户 CPU、系统 CPU、等待 IO CPU、软中断和硬中断等。

  - 用户 CPU 使用率，包括用户态 CPU 使用率（user）和低优先级用户态 CPU 使用率（nice），表示 CPU 在用户态运行的时间百分比，用户 CPU 使用率高，通常说明有应用程序比较繁忙
  - 系统 CPU 使用率，表示 CPU 在内核态运行的时间百分比（不包括中断），系统 CPU 使用率高，说明内核比较繁忙
  - 等待 IO CPU 使用率（iowait），表示等待 IO 的时间百分比，iowait 高，通常说明系统与硬件设备的 IO 交互时间比较长
  - 软中断和硬中断的 CPU 使用率，分别表示内核调用软中断处理程序、硬中断处理程序的时间百分比，他们的使用率高，通常说明系统发生了大量的中断
  - 虚拟化环境中用到的窃取 CPU 使用率（steal）、客户 CPU 使用率（guest），分别表示被其他虚拟机占用的 CPU 时间百分比和运行客户虚拟机的 CPU 时间百分比

- 平均负载（Load Average）

  系统的平均活跃进程数，它反应了系统的整体负载情况，主要包括三个数值，分别指过去 1 分钟、过去 5 分钟和过去 15 分钟的平均负载。

  理想情况下，平均负载等于逻辑 CPU 个数，表示每个 CPU 都恰好被充分利用。如果平均负载大于逻辑 CPU 个数，就表示负载比较重。

- 进程上下文切换

  上下文切换，本身是保证 linux 正常运行的一项核心功能。但过多的上下文切换，会将原本运行进程的 CPU 时间，消耗在寄存器、内核栈以及虚拟内存等数据的保存和恢复上，缩短进程真正运行的时间，成为性能瓶颈。

  - 无法获取资源而导致的资源上下文切换
  - 被系统强制调度导致的非自愿上下文切换

- CPU 缓存的命中率

  由于 CPU 发展的速度远快于内存的发展，CPU 的发展速度就比内存的访问速度快得多。CPU 在访问内存的时候，免不了要等待内存的响应。为了协调这两者之间的性能差距，CPU 缓存（通常是多级缓存）就出现了。

  CPU 缓存的速度介于 CPU 和内存之间，缓存的是热点的内存数据。按照大小不同分为 L1、L2、L3 等三级缓存，其中 L1、L2 常用在单核中，L3 则用在多核中。从 L1 到 L3 三级缓存的大小一次增大，相应地性能依次降低。它们的命中率，衡量的是 CPU 缓存的复用情况，命中率越高，表示性能越好。



### 根据指标找工具（CPU 性能）

| 性能指标            | 工具                                                 | 说明                                                         |
| ------------------- | ---------------------------------------------------- | ------------------------------------------------------------ |
| 平均负载            | uptime<br />top                                      | uptime 最简单<br />top提供了更全的指标                       |
| 系统整体 CPU 使用率 | vmstat<br />mpstat<br />top<br />sar<br />/proc/stat | top、vmstat、mpstat 只可以动态查看，而 sar 还可以记录历史数据<br />/proc/stat 是其他性能工具的数据来源 |
| 进程 CPU 使用率     | top<br />pidstat<br />ps<br />htop<br />atop         | top、ps 可以按 CPU 使用率给进程排序，而 pidstat 只显示实际用了 CPU 的进程<br />htop、atop以不同颜色显示更直观 |
| 系统上下文切换      | vmstat                                               | 除了上下文切换次数，还提供运行状态和不可中断状态进程的数量   |
| 进程上下文切换      | pidstat                                              | 注意加上 -w 选项                                             |
| 软中断              | top<br />/proc/softirqs<br />mpstat                  | top 提供软中断 CPU 使用率<br />/proc/softirqs、mpstat 提供了各种软中断在每个 CPU 上的运行次数 |
| 硬中断              | vmstat<br />/proc/interrupts                         | vmstat 提供总的中断次数<br />/proc/interrupts 提供各种中断在每个 CPU 上运行的累积次数 |
| 网络                | dstat<br />sar<br />tcpdump                          | dstat、sar 提供总的网络接收和发送情况<br />tcpdump 是动态抓取正在进行的网络通讯 |
| IO                  | dstat<br />sar                                       | 提供了 IO 的整体情况                                         |
| CPU 个数            | /proc/cpuinfo<br />lscpu                             | lscpu 更直观                                                 |
| 事件剖析            | perf<br />execsnoop                                  | perf 可以用来分析 CPU 的缓存以及内核调用链<br />execsnoop 用来监控短时进程 |



### 根据工具查指标（CPU 性能）

| 性能工具         | CPU 性能指标                                                 |
| ---------------- | ------------------------------------------------------------ |
| uptime           | 平均负载                                                     |
| top              | 平均负载、运行队列、整体的 CPU 使用率以及每个进程的状态和 CPU 使用率 |
| htop             | top 增强版，以不同颜色区分不同类型的进程，更直观             |
| atop             | CPU、内存、磁盘和网络等各种资源的全面监控                    |
| vmstat           | 系统整体的 CPU 使用率、上下文切换次数、中断次数，还包括处于运行和不可中断状态的进程数量 |
| mpstat           | 每个 CPU 的使用率和软中断次数                                |
| pidstat          | 进程和线程的 CPU 使用率、中断上下文切换次数                  |
| /proc/softirqs   | 软中断类型和在每个 CPU 上的累积中断次数                      |
| /proc/interrupts | 硬中断类型和在每个 CPU 上的累积中断次数                      |
| ps               | 每个进程状态和 CPU 使用率                                    |
| pstree           | 进程的父子关系                                               |
| dstat            | 系统整体的 CPU 使用率                                        |
| sar              | 系统整体的 CPU 使用率，包括可配置的历史数据                  |
| strace           | 进程的系统调用                                               |
| perf             | CPU 性能事件剖析，如调用链分析、CPU 缓存、CPU 调度等         |
| execsnoop        | 监控短时进程                                                 |



### 快速分析 CPU 性能瓶颈

top、vmstat、pidstat 分别提供了重要的 CPU 指标，虚线表示关联关系：

- 从 top 的输出可以查看各种 CPU 使用率以及僵尸进程和平均负载等信息
- 从 vmstat 的输出可以得到上下文切换次数、中断次数、运行状态和不可中断状态的进程数
- 从 pidstat 的输出可以得到进程的用户 CPU 使用率、系统 CPU 使用率、资源上下文切换、非资源上下文切换情况

![cpu_analyse](img/cpu_analyse.png)
