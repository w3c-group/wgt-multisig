# wgt-multisig

WGT 2 of 3 Multisig Server

机器人 Mixin ID: 7000102241

### 用途介绍

W3 协作组的平台通证为 WGT，10% 预挖 90% 通过积分挖得，而积分与贡献值对应。积分和 WGT 目前都在同一个机器人里发出，平台内对于每个用户都有一个内部代管的 Mixin 账号，用于存放未提现的 WGT，以及拿到的积分。

为保证 90%挖矿部分 WGT 按照约定规则释放，避免平台漏洞或其他风险导致经济模型崩溃，于是有了 wgt-multisig 这个多签服务，由 3 个节点完成 2/3 签名，实际运行时分类型每天进行预支申请，并非每次签到都进行申请。

### 运行逻辑

1.存放任意数额的 WGT

Deposit

2.发送挖矿请求，指定数量和账户

New Proposal

3.判断满足约定 rules 则通过签名并放币

Approve

4.重复 2 和 3

http://localhost:9300/proposal?rule=GrowthLimit&symbol=wgt&amount=1&userId=1b99263c-5223-42d3-82bc-637e68afc66a

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
