import { ServiceInfo } from '../../types';
import { Swagger } from '@atlassian/atlassian-openapi';

export abstract class GateProvider {
  abstract registerServices(
    services: {
      service: ServiceInfo;
      oas: Swagger.SwaggerV3 | undefined;
    }[],
  ): Promise<void>;
}
