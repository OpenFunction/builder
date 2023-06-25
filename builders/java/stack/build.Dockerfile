# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG from_image
FROM ${from_image}

ARG cnb_uid=${CNB_USER_ID}
ENV cnb_gid=${CNB_GROUP_ID}

ARG maven_version=3.8.8
ARG gradle_version=7.4.2

COPY licenses/ /usr/local/share/licenses/buildpacks/

# unzip is required to extract gradle.
RUN apt-get update && apt-get install -y --no-install-recommends \
  zip \
  unzip \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

ADD https://dlcdn.apache.org/maven/maven-3/${maven_version}/binaries/apache-maven-${maven_version}-bin.tar.gz /
ADD https://services.gradle.org/distributions/gradle-${gradle_version}-bin.zip  /

RUN mkdir /usr/local/maven && \
  tar xzf apache-maven-${maven_version}-bin.tar.gz -C /usr/local/maven && \
  rm -rf apache-maven-${maven_version}-bin.tar.gz && \
  ln -s /usr/local/maven/apache-maven-${maven_version} /usr/local/maven/current && \
  chown -R ${cnb_uid}:${cnb_gid} /usr/local/maven && \
  mkdir /usr/local/gradle && \
  unzip -q gradle-${gradle_version}-bin.zip -d /usr/local/gradle && \
  rm -rf gradle-${gradle_version}-bin.zip && \
  ln -s /usr/local/gradle/gradle-${gradle_version} /usr/local/gradle/current && \
  chown -R ${cnb_uid}:${cnb_gid} /usr/local/gradle

ENV PATH=$PATH:/usr/local/maven/current/bin:/usr/local/gradle/current/bin

USER cnb

