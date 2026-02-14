import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

import Docker from 'dockerode';
import { IDockerEvent, ServiceInfo } from '../types';

@Injectable()
export class DockerService {
  docker: Docker;
  networkName: string;

  unwatchedServices: string[];

  constructor(private readonly configService: ConfigService) {
    this.docker = new Docker({ socketPath: '/var/run/docker.sock' });
    this.networkName = this.configService.getOrThrow('APP_NAME');
    this.unwatchedServices = [
      'traefik',
      'keycloak',
      'krakend',
      'postgres',
      'registry',
    ].map(name => `${this.configService.get('APP_NAME')}_${name}`);
  }

  async watch(cb: () => Promise<void>) {
    const stream = await this.docker.getEvents({
      filters: { type: ['container'] },
    });
    stream.on('data', (chunk: Buffer) => {
      const event = JSON.parse(chunk.toString('utf8')) as IDockerEvent;
      const serviceName = event.Actor?.Attributes?.name;
      const eventAction = event.Action;
      if (
        serviceName &&
        eventAction &&
        !this.unwatchedServices.includes(serviceName) &&
        ['health_status: healthy'].includes(eventAction)
      )
        void cb();
    });
  }

  async listContainers(): Promise<ServiceInfo[]> {
    const containers = await this.docker.listContainers();

    const services: ServiceInfo[] = [];

    containers.forEach(c => {
      const network = c.NetworkSettings.Networks[this.networkName];
      const serviceName = c.Labels['infra.name'];
      if (network && serviceName && c.Labels['infra.enabled'] === 'true') {
        services.push({
          name: serviceName,
          ip: network.IPAddress,
          port: c.Labels['infra.port']
            ? parseInt(c.Labels['infra.port'])
            : 3000,
          version: c.Labels['infra.version'] ?? '1',
          oasEndpoint: c.Labels['infra.oas_endpoint'] ?? 'openapi',
        });
      }
    });

    return services;
  }
}
