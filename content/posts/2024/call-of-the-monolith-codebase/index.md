---
categories: ["article"]
tags: ["opinion", "monolith", "architecture", "microservices", "programming", "maintainability", "bazel"]
date: "2024-10-22T10:00:00Z"
description: "Are monoliths cool again?"
cover: "cover.jpg"
images: ["/posts/call-of-the-monolith-codebase/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "The Call of the Monolithic Codebase"
slug: "call-of-the-monolith-codebase"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/call-of-the-monolith-codebase/
mastodonID: "113350513645286780"
---

As the microservices landscape continues to evolve, a sobering reality is setting in for many organizations: managing many microservice repositories has become a significant hurdle. In the pursuit of scalability and flexibility offered by microservices, we may have inadvertently sacrificed maintainability, but now the pendulum is swinging back towards simpler code management while retaining the ability to deploy at scale. For organizations already entrenched in the microservices world, it's time to reconsider an old friend: the **monolithic codebase**.

Devs at these organizations might have noticed that as the number of services grow, so does the complexity of managing multiple codebases, each with its own repository, build pipelines, etc. This explosion of the number of repos introduces significant overhead, including:

- **Increased Operational Complexity:** More repositories mean more CI/CD pipelines, dependency management, and security audits.
- **Higher Cognitive Load for Developers:** Context switching between numerous repositories can decrease productivity. "Oh, this repo uses the old docker-compose scheme for testing, we should really remember update that."
- **Unified Change Management Challenges:** Implementing organization-wide changes, such as dependency updates or coding standards, becomes a monumental task.

### Scenario: Updating `log4j` across 20 Microservices Repositories
To further illustrate the challenges of managing multiple repositories in a microservices architecture, let's delve into the process of updating a commonly used library across all microservice repos.

- **Discovery:**
    - Identify the need to update `log4j` due to [a security vulnerability](https://blog.cloudflare.com/inside-the-log4j2-vulnerability-cve-2021-44228/). Maybe you hear about this from dependabot spamming you, maybe another security scanner picked up that your repo is using a log4j, maybe your devs see it at the top of hacker news or maybe you just have a grumpy security operations team reviewing every CVE and deciding if anything the company uses is vulnerable. Hopfully it's all of the above if the vulnerability is really severe.
    - Determine the versions currently in use across all 20 microservices.
- **Planning:**
    - Coordinate with multiple team leads to schedule the update. Depending on how "mono" the "monolith" is, you may have several teams sharing the same codebase.
    - Ensure compatibility with each microservice's specific dependencies.
- **Execution:**
    1. **Update Dependency:**
        - Modify the `pom.xml` (Maven) or `build.gradle` (Gradle) in each of the 20 repositories.
        - Commit changes with a consistent, informative message (e.g., "Security: Update log4j to v2.17.1").
    2. **Verify and Test:**
        - Run automated tests for each microservice to catch any breaking changes.
        - Perform manual testing where automated coverage is insufficient.
    3. **Deployment:**
        - Trigger CI/CD pipelines for each microservice to deploy the updated library.
- **Verification and Monitoring:**
    - Confirm the update has been successfully deployed across all microservices.
    - Monitor application logs for any issues related to the `log4j` update.

#### Challenges Encountered
Now this is the process if everything goes well. There's several different points where this process can increase the complexity of our simple task:

- **Version Inconsistencies:** Different microservices were using various versions of `log4j`, requiring tailored updates. Maybe some repos were on a previous major version and you'll need to update a lot of code in order to upgrade to the latest version. Or you may consider backporting the fix and maintaining patches or a fork of this repository.
- **Dependency Conflicts:** Updates in some microservices revealed hidden dependencies on outdated libraries. Maybe the latest version of log4j pulls in a new version of apache commons. Maybe another dependency, for some reason, doesn't work with the latest version. This kind of thing does happen and can be a real pain to resolve.

{{< image src="limes.png" width="500px" class="center" >}}

With all of this tedious work you should remember that this type of update is *the simplest kind of change to make*. Now just imagine trying to be proactive and keep all of your services up-to-date and to not let the least loved ones get left behind.

#### Monolithic Codebase Alternative
Because of the overhead in this process, some developers have leaned towards consolidating most of the code that powers microservices into a single repo. We refer to this type of code repository as a "monorepo". In our log4j scenario, it would help with several of these steps.

- **Single Update Point:** Modify the `log4j` dependency in one place, within the unified monorepo.
- **Simplified Testing and Deployment:** Leverage the monorepo's integrated testing and deployment mechanisms. If this is a commonly used library, more automated tests will need to be ran, but these are tests that would have ran anyway, but just spread out amongst many different repositories.
- **Reduced Complexity and Resource Usage:** Minimize the logistical overhead and focus on verifying the update's success.

## The Allure of the Monolithic Codebase
As demonstrated by the `log4j` update scenario, managing numerous repositories can be a complex and time-consuming process. A monolithic codebase offers a compelling alternative by simplifying many aspects of development and maintenance. Let's explore some of its key advantages:

- **Simplified Repository Management:** Instead of juggling 20 separate repositories, a monorepo consolidates everything into a single location. This significantly reduces the operational overhead associated with managing multiple codebases, build pipelines, and deployment processes. Imagine the time saved by not having to update `log4j` in 20 different places!
- **Unified Codebase Governance:** Implementing organization-wide changes, like security updates or coding standard enhancements, becomes significantly easier with a monorepo. A single update propagates across all services, ensuring consistency and reducing the risk of inconsistencies or oversights that can occur when managing multiple repositories.
- **Improved Developer Experience:** Developers can navigate the entire codebase more easily, reducing the cognitive load associated with context switching between different repositories. No more remembering which repo uses a specific Docker Compose scheme or wondering where a particular service is located. Everything is readily accessible in one place.
- **Enhanced Code Reusability:** With all services residing in a single repository, code sharing and reuse become more straightforward. Developers can easily identify and leverage existing components, reducing code duplication and promoting consistency across the application. This can be particularly beneficial for common libraries like `log4j`, where a single, well-maintained implementation can be used by all services.

Furthermore, a monorepo can significantly streamline testing and deployment processes:

- **Simplified Testing:** Automated tests can be run across all affected services with a single command, ensuring that changes don't introduce unexpected regressions. In the `log4j` scenario, this means verifying the update's impact on all services simultaneously, rather than testing each microservice individually.
- **Streamlined Deployment:** Atomic deployments become possible, guaranteeing that all services are updated consistently. This eliminates the risk of partial deployments or version mismatches that can occur in a microservices architecture.
- **Easier Rollbacks:** If an issue arises after deployment, rolling back to a previous version becomes simpler with a monorepo. A single rollback operation can revert all services to a known good state, minimizing downtime and disruption. With microservices that all have different versions, it's hard to know what a good combination of versions for each service actually is.

Note that while the codebase may be monolithic, many developer teams are still deploying as microservices. This promises the best of both worlds: the simplified and streamlined experience of a single repo with the scalability properties of microservices.

{{< image src="assimilate.png" width="500px" class="center" >}}

However, it's important to acknowledge that monolithic codebases are not without their challenges.

## Tooling for a Successful Monolithic Codebase
To leverage the benefits of a monolithic codebase while avoiding the pitfalls of tight coupling and decreased maintainability, consider the following tools:

- **[Bazel](https://bazel.build:/):** A build tool and ecosystem that supports large, multi-language monorepos; created by Google.
- **[Pants](https://www.pantsbuild.org:/):** A build system for monorepos, focusing on performance and scalability.
- **[Buck](https://buck2.build:/):** A fast and scalable build tool developed by Facebook, designed to handle large, complex monorepos with ease.
- **[Service Weaver](https://serviceweaver.dev/):** Service weaver has a very different take on monorepos. It's more than just a build tool, and is more of a framework that allows for monorepos with monolithic or microservice deployment. Because it is opinionated, other kinds of build tools aren't needed as much. Also, this is only supported for Go.
- **[Lerna](https://lerna.js.org/):** A popular and mature tool for managing JavaScript monorepos, especially useful for managing multiple packages within a single repository.
- **[Turbo](https://turbo.build/):** High-performance build system for JavaScript/TypeScript monorepos, known for its speed and efficient caching mechanisms.
- **[Git Submodules](https://git-scm.com/book/en/v2/Git-Tools-Submodules):** While not a dedicated monorepo tool, Git submodules can be used to manage dependencies on other Git repositories within a larger monorepo. However, they can be complex to use and may not be the best choice for all scenarios.
- **Your own tooling:** Sometimes building a set of scripts to customize the build process using more traditional tooling may be better.

You should note that this list is by no means complete. There are a lot of options (I think I found around 10 Javascript tools before giving up my research). I also left out several tools don't have open source tooling or rely exclusively on paid hosting services.

The choice of tooling will highly depend on the language/languages your codebases are hosted in. Bazel excels at seamlessly handling polyglot repos no matter the language. Others, like Pants only support a handful of languages and if you're using another language you may need to use multiple tools to get the job done. In general though, the fewer languages the easier it is to integrate because there are always language-specific idiosyncrasies that pop up, even when using tools like Bazel which try to hide those differences from you. Choose what makes sense for your team... and maybe try to get as far as you can using standard tooling before introducing extra build tools.

## Drawbacks of Monolithic Codebases
You should consider some of the real drawbacks when working with a monolithic codebase.

- **Increased Build Times:** As the codebase grows, build times can become significantly longer, especially if not properly optimized. This can slow down development and deployment cycles.
- **Merge Conflicts:** With many developers working on the same codebase, merge conflicts can become more frequent and complex, requiring more time and effort to resolve.
- **Tight Coupling:** If not carefully managed, a monorepo can lead to tight coupling between different parts of the application, making it harder to modify or refactor individual components without affecting others.
- **Learning Curve:** For developers accustomed to working with microservices and separate repositories, there can be a learning curve associated with navigating and understanding a large, monolithic codebase.
- **Access Control:** Managing access control and permissions can be more challenging in a monorepo, as developers potentially have access to the entire codebase, even if they only work on a specific part of it.

### Bazel-Specific Drawbacks
Since Bazel is a popular choice for managing monorepos, it's worth noting some of its specific drawbacks:

- **Steep Learning Curve:** Bazel has a reputation for having a steep learning curve, especially for developers unfamiliar with build systems like [Make](https://makefiletutorial.com/) or [CMake](https://cmake.org/). Its configuration language, Starlark, can also take time to master.
- **Limited IDE Support:** While IDE support for Bazel is improving, it's still not as comprehensive as for other build systems. This can make debugging and code navigation more challenging.
- **Performance Issues:** While Bazel is generally performant, it can sometimes experience performance issues, especially with very large codebases or complex build configurations.
- **Community Support:** While Bazel has a growing community, it's still smaller than the communities for other build systems like Maven, Gradle, or default tooling from languages like Rust, Go, etc. This can make it harder to find help or resources when encountering issues.

{{< image src="bazel.png" width="500px" class="center" >}}

## Mitigating the Drawbacks
It's important to note that many of these drawbacks can be mitigated with careful planning and the right tools and practices:

- **Modular Design:** Employing modular design principles and clear code organization can help minimize tight coupling and improve build times.
- **Code Ownership:** Establish clear code ownership and access control policies to manage permissions and prevent unauthorized modifications.
- **Continuous Integration and Continuous Deployment (CI/CD):** Implement robust CI/CD pipelines to automate builds, tests, and deployments, helping to identify and address issues early on.
- **Invest in Training:** Provide adequate training and support to help developers learn Bazel and its best practices.
- **Monitoring and Optimization:** Regularly monitor build times and Bazel's performance to identify and address any bottlenecks. Many of these build systems have more advanced features like remote caching and remote execution, which can be used to greatly reduce build times.

By acknowledging these potential drawbacks and taking steps to mitigate them, you can increase the likelihood of successfully adopting a monolithic codebase with Bazel.

## The Path Forward
While a monolithic codebase can simplify management and development, it's crucial to strike a balance between unity and the autonomy that microservices provide. Consider the following:

- **Start Small:** If migrating from a microservices architecture, begin by consolidating closely related services.
- **Establish Clear Boundaries:** Use tooling and guidelines to maintain logical separations within the monorepo.
- **Monitor and Adapt:** Continuously evaluate the effectiveness of your monolithic approach and make adjustments as needed.

As we've explored the trade-offs between separate repos and monolithic codebases, it's clear that the latter can offer a more streamlined development experience, simplified management, and reduced overhead. However, before embracing a monolithic codebase, it's essential to carefully consider the implications of this approach on your project's specific needs.
