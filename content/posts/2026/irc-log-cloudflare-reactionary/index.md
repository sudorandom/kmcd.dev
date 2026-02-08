---
title: "IRC Log: Reactionary"
date: 2026-03-02T10:00:00Z
cover: "cover.svg"
tags: [humor, irc, react, incident]
series: [irc-logs]
---

{{< chat >}}
{{< chat-server time="14:15:00" >}}PagerDutyBot: SEV-0: GLOBAL CONTROL PLANE UNREACHABLE. API ERROR RATE > 95%.{{< /chat-server >}}
{{< chat-action type="join" from="sev_manager" time="14:15:05" >}}{{< /chat-action >}}
{{< chat-msg from="sev_manager" time="14:15:10" >}}Status?{{< /chat-msg >}}
{{< chat-msg from="net_ops_jen" time="14:16:12" >}}It’s bad. Traffic to the auth service just verticalized. We’re seeing 50M RPS.{{< /chat-msg >}}
{{< chat-msg from="sev_manager" time="14:17:00" >}}DDoS? Scrubbing center active?{{< /chat-msg >}}
{{< chat-msg from="net_ops_jen" time="14:17:30" >}}That's the weird part. It's bypassing the WAF. It looks like legit traffic. TLS fingerprints are valid.{{< /chat-msg >}}
{{< chat-msg from="sev_manager" time="14:18:10" >}}Did someone let Tyler push a new WAF regex? Please tell me we didn't backpedal into 2019.{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:19:00" >}}I’m looking at the sample logs. These are all authenticated requests.{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:19:45" >}}They are all hitting `POST /api/v4/user/token/refresh`.{{< /chat-msg >}}
{{< chat-msg from="sev_manager" time="14:20:10" >}}Why is everyone refreshing their token at the exact same second?{{< /chat-msg >}}
{{< chat-msg from="net_ops_jen" time="14:21:00" >}}It’s not one refresh. I’m seeing the SAME user IDs hitting it 500 times per second.{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:22:15" >}}Uh, guys?{{< /chat-msg >}}
{{< chat-msg from="sev_manager" time="14:22:30" >}}Go ahead Felix.{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:23:00" >}}We pushed the "Seamless Session" update to the dashboard 15 minutes ago.{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:23:15" >}}The goal was to silently refresh the token in the background so users don't get logged out.{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:24:00" >}}Felix... look at the code.{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:25:00" >}}I am.{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:25:10" >}}...{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:25:15" >}}Felix?{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:25:45" >}}Oh no.{{< /chat-msg >}}
{{< chat-msg from="sev_manager" time="14:26:00" >}}REPORT.{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:26:30" >}}Okay, so, in React... `useEffect` runs when a dependency changes.{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:26:45" >}}We have: `useEffect(() => { refreshToken() }, [token])`{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:27:00" >}}And `refreshToken()`... updates the `token`?{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:27:05" >}}Yes.{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:27:10" >}}Which triggers the `useEffect` again?{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:27:12" >}}Yes.{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:27:30" >}}So you turned every single customer's browser tab into an infinite loop cannon pointed at our auth servers?{{< /chat-msg >}}
{{< chat-msg from="frontend_felix" time="14:28:00" >}}In my defense, the tokens are incredibly fresh.{{< /chat-msg >}}
{{< chat-msg from="sev_manager" time="14:28:15" >}}Rollback the frontend.{{< /chat-msg >}}
{{< chat-msg from="sev_manager" time="14:29:10" >}}I'll start writing another ridiculously well edited postmortem blog post.{{< /chat-msg >}}
{{< chat-msg from="backend_bob" time="14:29:41" >}}Hackernews and primeagen is going to going to love this one.{{< /chat-msg >}}
{{< /chat >}}

[Cloudflare outage on December 5, 2025](https://blog.cloudflare.com/5-december-2025-outage/)
