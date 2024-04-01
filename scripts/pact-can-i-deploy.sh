#!/bin/bash
# This script is invoked by cc-pact.mk target `pact-can-i-deploy`

set -u

# set +e allows describe-version to exit with error
# which we'll capture and print later
set +e
output=$("${PACT_BIN_PATH}"/pact-broker describe-version \
        --pacticipant="${PACTICIPANT_NAME}" \
        --version="${PACT_DEPLOY_VERSION}" \
        --broker-base-url="${PACT_BROKER_URL}" 2>&1)

exit_code=$?

# other errors are not allowed
set -e

if [ "${output}" = "Pacticipant version not found" ]; then
    # If describe version tells us this version is not found, it's ok, we can still allow deployment
    # This is to support e.g. rolling back to a version before pact was added
    echo "--- Pact Broker: ${PACTICIPANT_NAME} @ ${PACT_DEPLOY_VERSION} does not exist in the Pact Broker at ${PACT_BROKER_URL}, allowing deployment"
elif [ $exit_code -ne 0 ]; then
    # any other error is not ok
    echo -e "--- Pact Broker: describe-version command failed with exit code ${exit_code}, output:\n ${output}"
else
    echo "--- Pact Broker: can-i-deploy ${PACTICIPANT_NAME} @ ${PACT_DEPLOY_VERSION} to ${PACT_RELEASE_ENVIRONMENT}"
    command="${PACT_BIN_PATH}/pact-broker can-i-deploy \
			--pacticipant=${PACTICIPANT_NAME} \
			--version=${PACT_DEPLOY_VERSION} \
			--to-environment=${PACT_RELEASE_ENVIRONMENT} \
			--broker-base-url=${PACT_BROKER_URL}"
    if [ "${PACT_BROKER_CAN_I_DEPLOY_DRY_RUN}" != true ]; then
        # only use retry params when not in dry-run to avoid waiting too long for nothing.
        # https://github.com/pact-foundation/pact_broker-client/issues/149
        # Retry for 15 minutes to allow provider webhook job to finish.
        command="${command} --retry-while-unknown=30 --retry-interval=30"
    fi
    ${command}
    
fi
