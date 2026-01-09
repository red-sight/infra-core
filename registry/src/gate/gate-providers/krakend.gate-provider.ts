import { Swagger } from '@atlassian/atlassian-openapi';
import { ServiceInfo } from '../../types';
import { GateProvider } from './gate-provider';
import { writeFile } from 'node:fs/promises';
import {
  KrakendConfig,
  KrakendEndpoint,
  KrakendHttpMethod,
} from '../../types/krakend.types';

export class KrakendGateProvider extends GateProvider {
  async registerServices(
    services: {
      service: ServiceInfo;
      oas: Swagger.SwaggerV3 | undefined;
    }[],
  ): Promise<void> {
    const krakendConfig: KrakendConfig = {
      $schema: 'https://www.krakend.io/schema/v2.12/krakend.json',
      version: 3,
      endpoints: [],
    };

    services.forEach(({ service, oas }) => {
      if (!oas) return;

      const host = `http://${service.ip}:${service.port}`;

      Object.keys(oas.paths).forEach(path => {
        const endpoint = `/${service.name}${path}`;

        Object.keys(oas.paths[path]).forEach(m => {
          const method = m.toUpperCase() as KrakendHttpMethod;

          const existingEndpoint = krakendConfig.endpoints.find(
            e =>
              e.endpoint === endpoint &&
              (e.method.toUpperCase() as KrakendHttpMethod) === method,
          );

          const existingHosts = existingEndpoint?.backend[0].host ?? [];
          const updatedHost = [...new Set([...existingHosts, host])];

          const endpointConfig: KrakendEndpoint = {
            endpoint,
            method,
            backend: [
              {
                url_pattern: path,
                host: updatedHost,
                method,
              },
            ],
          };

          if (existingEndpoint) Object.assign(existingEndpoint, endpointConfig);
          else krakendConfig.endpoints.push(endpointConfig);
        });
      });
    });

    await writeFile(
      '/krakend/krakend.json',
      JSON.stringify(krakendConfig, null, 2),
    );
  }
}
