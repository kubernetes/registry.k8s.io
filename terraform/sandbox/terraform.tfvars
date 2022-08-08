/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

tag        = "v20220331-v0.0.1-4-gc3b27f3"
domain     = "registry-sandbox.k8s.io"
project_id = "k8s-infra-oci-proxy"
cloud_run_config = {
  asia-east1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://asia.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  asia-southeast1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://asia.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  asia-southeast2 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://asia.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  europe-north1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://eu.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  europe-southwest1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://eu.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
 europe-west1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://eu.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  europe-west4 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://eu.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  europe-west8 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://eu.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  europe-west9 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://eu.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  us-central1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://us.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  us-east1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://us.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  us-east4 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://us.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  us-east5 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://us.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  us-south1 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://us.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
  us-west2 = {
    environment_variables = [
      {
        name  = "UPSTREAM_REGISTRY_ENDPOINT",
        value = "https://us.gcr.io"
      },
      {
        name  = "UPSTREAM_REGISTRY_PATH",
        value = "k8s-artifacts-prod"
      }
    ]
  }
}
