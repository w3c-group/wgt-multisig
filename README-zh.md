# wgt-multisig
WGT 2 of 3 Multisig Server


### 用途介绍

W3协作组的平台通证为WGT，10%预挖90%通过积分挖得，而积分与贡献值对应。积分和WGT目前都在同一个机器人里发出，平台内对于每个用户都有一个内部代管的Mixin账号，用于存放未提现的WGT，以及拿到的积分。

为保证90%挖矿部分WGT按照约定规则释放，避免平台漏洞或其他风险导致经济模型崩溃，于是有了wgt-multisig这个多签服务，由3个节点完成2/3签名，每次签到都会提交请求给节点，验证账号昨日持有的积分，以及对比积分增速是否正常等规则，最终通过并将当前签到领取的挖矿WGT收益，在Mixin Network中转帐到用户对应的在平台的代管账号中。

可以看到自己节点的统计数据，支持随机抽查用户的积分获取明细，会统计更详细的积分持有和新增情况。


### 运行逻辑

1.将WGT所有币转到多签，共3个节点，其中1个Master节点。

2.Master节点随机请求其他任一节点，完成多签验证有效性之后，根据积分持有情况发放WGT，释放WGT到指定账户。

3.统计积分持有增加情况，防止积分量异常增加的Bug，对于异常账号会在Master节点就过滤掉，其余如果有疏漏可以由另外2个节点质疑并拒绝释放WGT。

4.如果暂时不能完成多签，签到时会提示稍后并进入重试队列，也就是2个节点离线超过一天需要次日补发。


### 启动方式

使用 systemctl 运行:

```
vi /etc/systemd/system/wgt-multisig.service
```


配置：

```
[Unit]
Description=wgt-multisig
Documentation=https://w3c.group
After=network.target

[Service]
User=root
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity

ExecStart=/root/wgt-multisig/wgt-multisig
Restart=on-failure

[Install]
WantedBy=multi-user.target
```


运行及查看日志：

```
systemctl daemon-reload
systemctl status wgt-multisig
systemctl start wgt-multisig

journalctl -f -u wgt-multisig.service
```
