---
title: "IRC Log: rm -rf"
date: 2026-03-09T10:00:00Z
cover: "cover.svg"
tags: [humor, irc, rm, incident]
series: [irc-logs]
---

# The Incident Log: January 31, 2017

{{< chat >}}
{{< chat-server time="23:00" >}}Topic: DB Replication Lag | Status: 🔴 Critical{{< /chat-server >}}
{{< chat-action type="join" from="tired_sysadmin" time="23:05" >}}{{< /chat-action >}}
{{< chat-msg from="tired_sysadmin" time="23:10" >}}Replication is stuck again. The secondary node (db2) is refusing to sync.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:11" >}}I'm going to wipe the data directory on db2 and let it pull a fresh copy from master.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:12" >}}rm -rf /var/opt/gitlab/postgresql/data{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:12" >}}Weird. It's taking a while. Usually empty directories delete instantly.{{< /chat-msg >}}
{{< chat-msg from="helper_dev" time="23:13" >}}Hey, why did the website just go 500?{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:13" >}}...{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:13" >}}I'm looking at my terminal prompt.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:14" >}}It says root@db1.{{< /chat-msg >}}
{{< chat-msg from="helper_dev" time="23:14" >}}db1 is Prod. You are deleting Prod.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:15" >}}CTRL+C CTRL+C CTRL+C{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:15" >}}Okay, I stopped it. How much is left?{{< /chat-msg >}}
{{< chat-msg from="helper_dev" time="23:16" >}}Checking... The directory is 4.5KB.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:16" >}}We had 300GB of data.{{< /chat-msg >}}
{{< chat-msg from="helper_dev" time="23:17" >}}Okay, don't panic. We have 5 different backup mechanisms. Let's check S3.{{< /chat-msg >}}
{{< chat-msg from="helper_dev" time="23:20" >}}S3 bucket is empty. The backup script has been failing silently since version 8.1.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:21" >}}Check the Azure disk snapshots.{{< /chat-msg >}}
{{< chat-msg from="helper_dev" time="23:22" >}}Not enabled.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:23" >}}...LVM snapshots?{{< /chat-msg >}}
{{< chat-msg from="helper_dev" time="23:24" >}}We take them every 24 hours. We just lost 6 hours of data.{{< /chat-msg >}}
{{< chat-msg from="tired_sysadmin" time="23:25" >}}I am going to live stream the restoration on YouTube so people don't kill us.{{< /chat-msg >}}
{{< /chat >}}

[Postmortem of database outage of January 31](https://about.gitlab.com/blog/postmortem-of-database-outage-of-january-31/)
