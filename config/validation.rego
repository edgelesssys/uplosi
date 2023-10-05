package config

import future.keywords.in

deny[msg] {
    not input.Provider in [ "aws", "azure", "gcp" ]

    msg = sprintf("cloud provider %s unknown", [input.Provider])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SubscriptionID == ""

    msg = "required field subscriptionID empty for provider azure"
}
