import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";

import Docker from "dockerode";
import { ServiceInfo } from "../types";

@Injectable()
export class DockerService {
  docker: Docker;
  networkName: string;

  constructor(private readonly configService: ConfigService) {
    this.docker = new Docker({ socketPath: "/var/run/docker.sock" });
    this.networkName = this.configService.getOrThrow("APP_NAME");
  }

  // async watch() {
  //   const stream = await this.docker.getEvents();
  //   stream.on('data', (chunk) => {
  //     const event = JSON.parse(chunk.toString('utf8'));
  //     console.log('Docker event:', event);
  //   });
  // }

  async listContainers(): Promise<ServiceInfo[]> {
    const containers = await this.docker.listContainers();

    const services: ServiceInfo[] = [];

    containers.forEach(c => {
      const network = c.NetworkSettings.Networks[this.networkName];
      const serviceName = c.Labels["infra.name"];
      if (network && serviceName && c.Labels["infra.enabled"] === "true") {
        services.push({
          name: serviceName,
          ip: network.IPAddress,
          port: c.Labels["infra.port"]
            ? parseInt(c.Labels["infra.port"])
            : 3000,
          version: c.Labels["infra.version"] ?? "1",
          oasEndpoint: c.Labels["infra.oas_endpoint"] ?? "openapi",
        });
      }
    });

    return services;
  }
}
