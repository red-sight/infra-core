import { Injectable } from "@nestjs/common";
import { ContainerInfo } from "dockerode";
import { DockerService } from "../docker/docker.service";
import { OpenapiService } from "../openapi/openapi.service";

@Injectable()
export class MainService {
  constructor(
    private readonly dockerService: DockerService,
    private readonly openapiService: OpenapiService,
  ) {}

  async listAllServices() {
    const containers = await this.dockerService.listContainers();

    for (const c of containers) {
      await this.openapiService.fetchSpec(c);
    }

    return containers;
  }

  registerService(service: ContainerInfo) {
    console.log(service);
  }
}
