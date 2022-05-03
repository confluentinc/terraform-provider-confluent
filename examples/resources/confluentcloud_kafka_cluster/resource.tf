resource "confluent_environment" "test-env" {
  display_name = "Development"
}

resource "confluent_kafka_cluster" "basic-cluster-on-aws" {
  display_name = "basic_kafka_cluster_on_aws"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-west-2"
  basic {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "basic-cluster-on-azure" {
  display_name = "basic_kafka_cluster_on_azure"
  availability = "SINGLE_ZONE"
  cloud        = "AZURE"
  region       = "centralus"
  basic {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "basic-cluster-on-gcp" {
  display_name = "basic_kafka_cluster_on_gcp"
  availability = "SINGLE_ZONE"
  cloud        = "GCP"
  region       = "us-central1"
  basic {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "standard-cluster-on-aws" {
  display_name = "standard_kafka_cluster_on_aws"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-west-2"
  standard {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "standard-cluster-on-azure" {
  display_name = "standard_kafka_cluster_on_azure"
  availability = "SINGLE_ZONE"
  cloud        = "AZURE"
  region       = "centralus"
  standard {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "standard-cluster-on-gcp" {
  display_name = "standard_kafka_cluster_on_gcp"
  availability = "SINGLE_ZONE"
  cloud        = "GCP"
  region       = "us-central1"
  standard {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "dedicated-cluster-on-aws" {
  display_name = "dedicated_kafka_cluster_on_aws"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-west-2"
  dedicated {
    cku = 1
  }

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "dedicated-cluster-on-azure" {
  display_name = "dedicated_kafka_cluster_on_azure"
  availability = "SINGLE_ZONE"
  cloud        = "AZURE"
  region       = "centralus"
  dedicated {
    cku = 1
  }

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_cluster" "dedicated-cluster-on-gcp" {
  display_name = "dedicated_kafka_cluster_on_gcp"
  availability = "SINGLE_ZONE"
  cloud        = "GCP"
  region       = "us-central1"
  dedicated {
    cku = 1
  }

  environment {
    id = confluent_environment.test-env.id
  }
}
