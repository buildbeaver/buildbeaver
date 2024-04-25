import logging
import client
from client.apis.tags import runners_api
from client.model.create_runner_request import CreateRunnerRequest
from client.paths.legal_entities_legal_entity_id_runners.post import RequestPathParams

logger = logging.getLogger(__name__)


class BBAPIClient:
    """
    Provides a wrapper around our generated Python core code for our E2E tests.
    """

    def __init__(self, api_url, auth_token):
        logger.info("Creating APIClient with api_url of %s", api_url)
        self.configuration = client.Configuration(
            host=api_url
        )
        # Configure API key authorization: secret_token
        self.auth_token = auth_token

    def register_runner(self, legal_entity_id, runner_name, runner_certificate_pem):
        """
        Registers a given runner against a legal entity id by its certificate (in PEM format)
        :param legal_entity_id: The ID of the Legal Entity to register the runner against.
        :param runner_name: The name of the runner to register.
        :param runner_certificate_pem: The certificate of the runner (in PEM format)
        :return:
        """
        # Enter a context with an instance of the API client
        with client.ApiClient(self.configuration, 'buildbeaver-token', self.auth_token) as api_client:
            api_instance = runners_api.RunnersApi(api_client)

        # CreateRunnerRequest | Runner registration information, used to submit a request to create a new runner.
        create_runner_request = CreateRunnerRequest(
            name=runner_name,
            client_certificate_pem=runner_certificate_pem,
        )

        path_parameters = RequestPathParams(legalEntityId=legal_entity_id)

        try:
            # Registers a new runner for a legal entity.
            api_response = api_instance.create_runner(create_runner_request, "application/json", path_parameters)
            logger.info("Runner registered with response: %s", api_response)
            return api_response
        except Exception as e:
            logger.error("Exception when calling RunnersApi->create_runner: %s", e)
            raise e
