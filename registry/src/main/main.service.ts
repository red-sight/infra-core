import { Injectable } from '@nestjs/common';
import { DockerService } from '../docker/docker.service';
import { OpenapiService } from '../openapi/openapi.service';
import { GateService } from '../gate/gate.service';

@Injectable()
export class MainService {
  constructor(
    private readonly dockerService: DockerService,
    private readonly openapiService: OpenapiService,
    private readonly gateService: GateService,
  ) {}

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
  }
}
