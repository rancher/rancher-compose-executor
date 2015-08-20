# Rancher Compose Executor [![Build Status](http://ci.rancher.io/api/badge/github.com/rancher/rancher-compose-executor/status.svg?branch=master)](http://ci.rancher.io/github.com/rancher/rancher-compose-executor)
--------------------------

This microservice will execute rancher-compose cli commands using appropriate credentials when stacks are created by pasting the `docker-compose.yml` and `rancher-compose.yml` files in the Rancher UI.

It is an [external event handler](https://github.com/rancher/cattle/blob/master/docs/examples/handler-bash/simple_handler.sh) in Rancher that listens for events related to the life cycle of ``Stacks`` resources. In this context, ``Stacks`` are called `environment` in the resource API

The following is the only event this event handler listens on

* ```environment.create```

# Contact
For bugs, questions, comments, corrections, suggestions, etc., open an issue in
 [rancher/rancher](//github.com/rancher/rancher/issues) with a title starting with `[rancher-compose-executor] `.

 Or just [click here](//github.com/rancher/rancher/issues/new?title=%5Brancher-compose-executor%5D%20) to create a new issue.

# License
Copyright (c) 2014-2015 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

