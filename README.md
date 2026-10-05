<!-- Sonar Marketing hosts these approved brand assets on its Kentico Kontent CDN (assets-eu-01.kc-usercontent.com). Shared URLs are intentional; consult Marketing before replacing them. -->
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://assets-eu-01.kc-usercontent.com/ef593040-b591-0198-9506-ed88b30bc023/a23fc7ba-23f0-489a-829d-ed88c0748521/Sonar_Logo_Dark%20Backgrounds.svg">
    <img src="https://assets-eu-01.kc-usercontent.com/ef593040-b591-0198-9506-ed88b30bc023/82c13eba-d95c-4bb8-8007-7ce77c14e043/Sonar_Logo_Light%20Backgrounds.svg" alt="Sonar logo" width="400">
  </picture>
</p>

[![Build](https://github.com/SonarSource/helm-chart-sonarqube/actions/workflows/build.yml/badge.svg)](https://github.com/SonarSource/helm-chart-sonarqube/actions/workflows/build.yml) [![Quality Gate Status](https://next.sonarqube.com/sonarqube/api/project_badges/measure?project=SonarSource_helm-chart-sonarqube&metric=alert_status&token=sqb_8b5471dd501cf438b25e938f50307572132b3640)](https://next.sonarqube.com/sonarqube/dashboard?id=SonarSource_helm-chart-sonarqube)

<!-- sonar-marketing:start -->
<!-- Marketing maintains this section. For wording changes, consult the relevant Product Marketing Manager (PMM). Repository CODEOWNERS review accuracy and merge changes. -->

# SonarQube Helm Chart

This repository contains the official Helm charts for deploying SonarQube Server and SonarQube Community Build on Kubernetes.

For Data Center Edition, use the [`charts/sonarqube-dce`](charts/sonarqube-dce) chart. Learn more on the [SonarQube Server product page](https://www.sonarsource.com/products/sonarqube/server/).

<!-- sonar-marketing:end -->

The actual chart can be found in the [charts](charts/sonarqube) directory and see the README of the chart for more information.

Have Questions or Feedback?
---------------------------

For support questions ("How do I?", "I got this error, why?", ...), please first read the [documentation](https://docs.sonarqube.org) and then head to the [SonarSource Community](https://community.sonarsource.com/c/help/sq/10). The answer to your question has likely already been answered! 🤓

Be aware that this forum is a community, so the standard pleasantries ("Hi", "Thanks", ...) are expected. And if you don't get an answer to your thread, you should sit on your hands for at least three days before bumping it. Operators are not standing by. 😄

Contributing
------------

If you would like to see a new feature, please create a new Community thread: ["Suggest new features"](https://community.sonarsource.com/c/suggestions/features). Please be aware that you must [create an account](https://community.sonarsource.com/signup) before you can suggest a new feature or browse through the existing suggestions.

Please be aware that we are not actively looking for feature contributions. The truth is that it's extremely difficult for someone outside SonarSource to comply with our roadmap and expectations. Therefore, we typically only accept minor cosmetic changes and typo fixes.

With that in mind, if you would like to submit a code contribution, please create a pull request for this repository. Please explain your motives to contribute to this change: what problem you are trying to fix, and what improvement you are trying to make.

> Our merging process involves duplicating pull requests, aligning the code with our standards, and conducting internal tests. You will be informed of this process by the creation of a Jira ticket. Please note that we implement these features on a best-effort basis, and therefore, we cannot provide an estimated time of arrival (ETA). The original author will still be attributed to all commits.

Willing to contribute to SonarSource products? We are looking for smart, passionate, and skilled people to help us build world-class code-quality solutions. Have a look at our current [job offers here](https://www.sonarsource.com/company/jobs/)!

Note of Thanks
--------------

This chart was based on the great work done on the [Oteemo chart](https://github.com/Oteemo/charts/tree/master/charts/sonarqube).
We would like to thank everyone who contributed to their great work on this project.

License
-------

Licensed under the [MIT Licence](LICENSE)
