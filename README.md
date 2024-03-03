# ginkgoless-kubebuilder

Example controller demonstrating kubebuilder-generated controller tests w/o ginkgo.

## Description

kubebuilder includes ginkgo in their scaffold, rendering it the de-facto standard testing framework for kubernetes controllers.
This repo intends to demonstrate on how to use kubebuilder w/o ginkgo/gomega for testing.

The [example](./example) is based on a quick [kubebuilder bootsrap](https://book.kubebuilder.io/quick-start.html#create-a-project):

```bash
kubebuilder init --domain my.domain --repo my.domain/guestbook
kubebuilder create api --group webapp --version v1 --kind Guestbook
make manifests
```

And afterwards replaces ginkgo/gomega from the [controller tests](./example/internal/controller/guestbook_controller_test.go) and the [e2e tests](./example/test/e2e/e2e_test.go).

## Getting Started

If you want to use this pattern in your controller:

- Make the [envtesthelper](./envtesthelper/envtesthelper.go) available in your project (either copypaste the gist, or add the [module](./envtesthelper/go.mod) as a dependency)
- Write your test as a simple table test, see [example](./example/internal/controller/guestbook_controller_test.go)

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

