import { Injectable, OnModuleDestroy } from '@nestjs/common';
import { Readable } from 'stream';
import { DockerService } from '../docker/docker.service';
import { OpenapiService } from '../openapi/openapi.service';
import { GateService } from '../gate/gate.service';

@Injectable()
export class MainService implements OnModuleDestroy {
  private watchStream: Readable | null = null;

  constructor(
    private readonly dockerService: DockerService,
    private readonly openapiService: OpenapiService,
    private readonly gateService: GateService,
  ) {}

  onModuleDestroy() {
    this.destroyWatcher();
  }

  private destroyWatcher() {
    if (this.watchStream) {
      this.watchStream.removeAllListeners();
      this.watchStream.destroy?.();
      this.watchStream = null;
    }
  }

  async listAllServices() {
    const containers = await this.dockerService.listContainers();

    for (const c of containers) {
      await this.openapiService.fetchSpec(c);
    }

    return containers;
  }

  async bootstrap() {
    const servicesInfo = await this.dockerService.listContainers();

    const services = await Promise.all(
      servicesInfo.map(async i => ({
        service: i,
        oas: await this.openapiService.fetchSpec(i),
      })),
    );

    await this.gateService.provider.registerServices(services);

    this.destroyWatcher();

    this.watchStream = (await this.dockerService.watch()) as Readable;
    this.watchStream.on('healthy', serviceName => {
      console.log(
        `1. Discovered service ${serviceName} instance start, listing containers...`,
      );

      void this.bootstrap();
    });
  }
}
