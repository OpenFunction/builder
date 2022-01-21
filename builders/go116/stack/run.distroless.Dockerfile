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

FROM openfunctiondev/distroless-static:nonroot 

# The uid and gid for a distroless nonroot user is 65532 
# according to https://github.com/GoogleContainerTools/distroless/issues/443
ARG cnb_uid=65532
ARG cnb_gid=65532
ARG stack_id="openfunction.go116"
LABEL io.buildpacks.stack.id=${stack_id}

ENV CNB_USER_ID=${cnb_uid}
ENV CNB_GROUP_ID=${cnb_gid}
ENV CNB_STACK_ID=${stack_id}