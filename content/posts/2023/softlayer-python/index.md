---
categories: ["article", "project"]
tags: ["python", "cloud", "softlayer", "api", "cli", "open-source"]
date: "2023-07-31"
description: "I wrote and maintained language bindings for a large cloud company. Join me as I reflect on that experience."
cover: "cover.jpg"
images: ["/posts/softlayer-python/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "softlayer-python: language bindings/CLI for a cloud company"
slug: "softlayer-python"
type: "posts"
devtoSkip: true
mastodonID: "112277298441633204"
---

I used to work for a public cloud company called [SoftLayer](https://en.wikipedia.org/wiki/IBM_Cloud#SoftLayer). As a cloud company, there is an API that customers can use to provision new virtual servers, load balancers, firewalls and whatever else you might want. On our team, we used SoftLayer services as a customer might and we ended up proving new products and just... experiencing what it was like as a customer. I loved the concept. Our team heavily used this practice of so-called "eating your own dog food."

When dogfooding our services it became painfully obvious that our API was extremely hard to use. When making tooling we had to go through extremely complex ordering APIs that were designed for internal use and were exposed publicly for convenience. I preferred to do this work in Python at the time, so I used our public API bindings. This was the [current state of the project on github](https://github.com/softlayer/softlayer-python/tree/59b331dd9c33d9582d425be192fd3c2d63368d5d). Let me point out some of the issues I had:

- Didn't work with Python 3.
- Used class variables incorrectly, making it impossible to use multiple instances of the client at the same time.
- Had an awkward use of dictionary accessors that made things more confusing as a user.
- Had absolutely zero unit tests.
- The Python module name used upper-case characters.

So... Immediately, I wanted to improve things. However, this was the first open-source project that I would modify that had a non-trivial number of users. Because of this, I learned to make strategic changes that followed the pattern of:

- Add several unit tests for the part of the code that
- Add a new interface that
- Use the new interface in the implementation of the old interface (to reduce code duplication)
- Add deprecation warnings for the old interface.
- After enough time, bump the major version of the library and remove all code that is deprecated.

This pattern can be incredibly tedious. However, if you are maintaining a project used by many then you need to worry about upgrading. You need to write release notes. You need to give enough examples. You need to provide an upgrade path when you want to make breaking changes. This is super boring work, but it's the difference between "when I upgrade this package everything breaks" and "when I upgrade this package, I get more cool new stuff".

For perspective, [here's what the code looks like now](https://github.com/softlayer/softlayer-python). Note that there are now almost 2,000 unit tests. Note that the tests are run with several different versions of Python. And you should also note that there's an entire command line client in the repo as well!

The CLI was also created from the same motivations. For us, it makes automating and testing things much easier. Instead of clicking through a virtual machine creation web form for 20 minutes I could just copy/paste a command that I've run before that could specify everything I would have to type or select anyway. In fact, here's an example of that!

```bash
$ slcli vs create --hostname=example --domain=softlayer.com -f B1_1X2X25 -o DEBIAN_LATEST_64  --datacenter=ams01 --billing=hourly
This action will incur charges on your account. Continue? [y/N]: y
    :..........:.................................:......................................:...........................:
    :    ID    :               FQDN              :                 guid                 :         Order Date        :
    :..........:.................................:......................................:...........................:
    : 70112999 : testtesttest.test.com : 1abc7afb-9618-4835-89c9-586f3711d8ea : 2019-01-30T17:16:58-06:00 :
    :..........:.................................:......................................:...........................:
    :.........................................................................:
    :                            OrderId: 12345678                            :
    :.......:.................................................................:
    :  Cost : Description                                                     :
    :.......:.................................................................:
    :   0.0 : Debian GNU/Linux 9.x Stretch/Stable - Minimal Install (64 bit)  :
    :   0.0 : 25 GB (SAN)                                                     :
    :   0.0 : Reboot / Remote Console                                         :
    :   0.0 : 100 Mbps Public & Private Network Uplinks                       :
    :   0.0 : 0 GB Bandwidth Allotment                                        :
    :   0.0 : 1 IP Address                                                    :
    :   0.0 : Host Ping and TCP Service Monitoring                            :
    :   0.0 : Email and Ticket                                                :
    :   0.0 : Automated Reboot from Monitoring                                :
    :   0.0 : Unlimited SSL VPN Users & 1 PPTP VPN User per account           :
    :   0.0 : 2 GB                                                            :
    :   0.0 : 1 x 2.0 GHz or higher Core                                      :
    : 0.000 : Total hourly cost                                               :
    :.......:.................................................................:
```

Here we're creating a VM inside of the Amsterdam data center that uses the latest Debian image with 2GB of RAM, a single CPU core and 25GB of disk space. All by copying a single command. This is super powerful. If you're wondering why everything is free it's because our account had special billing. 😛

After you create a VM, you can also list the running instances to see it:
```bash
$ slcli vs list
:.........:............:....................:.......:........:................:..............:....................:
:    id   : datacenter :       host         : cores : memory :   primary_ip   :  backend_ip  : active_transaction :
:.........:............:....................:.......:........:................:..............:....................:
: 1234567 :   sjc01    :  test.example.com  :   4   :   4G   :    12.34.56    :   65.43.21   :         -          :
:.........:............:....................:.......:........:................:..............:....................:
```

The story is the same for several other products. Here's a list of what products are supported today:

- Account Management
- Block Storage
- Bandwidth Pools
- CDN
- Dedicated Hosts
- DNS
- Email
- File Storage
- Firewall
- Global IP
- Dedicated Hardware
- Disk Images
- IPSec
- Licenses
- Load Balancers
- NAS
- Object Storage
- Ordering/Quotes
- SSH Keys and Certificates
- Security Groups
- Subnets
- Support Tickets
- Users
- VLANs
- Virtual Servers

It's remarkable how far this project has come. This started as a way to create some VMs with a script but it became an extremely important interface for the entire suite of cloud products. Slowly, the CLI grew to what it is today. This wasn't a result of just me. Instead, several members of mine and other people's teams contributed to the project. There were periods when I didn't write a lot of code but would spend most of my time reviewing and guiding others to make their contributions. I learned a lot about the importance of enforcing a single style and keeping the quality of code high. Essentially, the ways customers interfaced was through the website, API or the CLI.

The CLI also drove more usage of the Python client. It encouraged this growth in several ways:

- It showcased what was possible with the API in a way that the documentation just can't do.
- It acted as a good reference for "how can I do this with the API". The more we added to the CLI, the fewer extra examples we needed to make. It is very important that the CLI was also open source for this reason.
- It had a verbose flag that showed the API calls that were being made, greatly increasing visibility into how it works.

Here's an example of what a command looks like when running a command using the verbose flag.
```bash
$ slcli -v vs detail 74397127
Calling: SoftLayer_Virtual_Guest::getObject(id=74397127, mask='id,globalIdentifier,fullyQualifiedDomainName,hostname,domain', filter='None', args=(), limit=None, offset=None))
Calling: SoftLayer_Virtual_Guest::getReverseDomainRecords(id=77460683, mask='', filter='None', args=(), limit=None, offset=None))
:..................:..............................................................:
:       name       :                            value                             :
:..................:..............................................................:
:  execution_time  :                          2.020334s                           :
:    api_calls     :        SoftLayer_Virtual_Guest::getObject (1.515583s)        :
:                  : SoftLayer_Virtual_Guest::getReverseDomainRecords (0.494480s) :
:     version      :                   softlayer-python/v5.7.2                    :
:  python_version  :           3.7.3 (default, Mar 27 2019, 09:23:15)             :
:                  :              [Clang 10.0.1 (clang-1001.0.46.3)]              :
: library_location : /Users/chris/Code/py3/lib/python3.7/site-packages/SoftLayer  :
:..................:..............................................................:
```

The more `v` characters you add, the more verbose the output gets. If you use `-vvv` then you will get the equivalent cURL commands to make the same API calls, which should be clear enough for any developer to make a client against the API.

```bash
$ slcli -vvv account summary
curl -u $SL_USER:$SL_APIKEY -X GET -H "Accept: */*" -H "Accept-Encoding: gzip, deflate, compress"  'https://api.softlayer.com/rest/v3.1/SoftLayer_Account/getObject.json?objectMask=mask%5B%0A++++++++++++nextInvoiceTotalAmount%2C%0A++++++++++++pendingInvoice%5BinvoiceTotalAmount%5D%2C%0A++++++++++++blockDeviceTemplateGroupCount%2C%0A++++++++++++dedicatedHostCount%2C%0A++++++++++++domainCount%2C%0A++++++++++++hardwareCount%2C%0A++++++++++++networkStorageCount%2C%0A++++++++++++openTicketCount%2C%0A++++++++++++networkVlanCount%2C%0A++++++++++++subnetCount%2C%0A++++++++++++userCount%2C%0A++++++++++++virtualGuestCount%0A++++++++++++%5D'
```

In summary, this was an incredibly successful side project. What started as a small script for internal use turned into a Swiss army knife that was a completely new way to access all products that SoftLayer offered. I learned so much about maintaining an open-source project, choosing reliable libraries to build on, code quality/style, and so much more.

References:
- Github: https://github.com/softlayer/softlayer-python
- Documentation: https://softlayer-python.readthedocs.io/en/latest/
