# dbmotion_tool
创建SSH隧道,使squids用户能够访问内网数据库

# 使用方法
1. 检查ssh配置, 如果检查成功, 可以直接进行步骤3, 否则通过步骤2进行设置
   * 检查成功
       ```shell
       ../dbmotion_tool -t check
       sh config is ok: GatewayPorts yes
       GatewayPorts check ok
       ```
   * 检查失败
       ```shell
       GatewayPorts not found
       GatewayPorts check failed
       ```
2. 设置ssh, 开启GatewayPorts
       ```shell
       ./dbmotion_too -t set
       Check and set sshd.
       GatewayPorts not found, add it
       Restart sshd
       ```

3. 创建ssh反向隧道
       ```shell
       ./dbmotion_tool -t create -h 192.168.2.104 -p 13306
       create tunnel for 192.168.2.104:13306 on 48834
       tunnel for 192.168.2.104:13306 on 48834 is created.
      ```

4. 测试连接
      ```shell
       mysql -h dbmotion.squids.cn -p 48834 -u root -p
      ```

5. 关闭反向隧道
      ```shell
      ./dbmotion_tool -t close -p 48834
      ```
