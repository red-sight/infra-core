import { Module, OnModuleInit } from "@nestjs/common";
import { MainService } from "./main.service";
import { DockerModule } from "../docker/docker.module";
import { OpenapiModule } from "../openapi/openapi.module";

@Module({
  providers: [MainService],
  imports: [DockerModule, OpenapiModule],
})
export class MainModule implements OnModuleInit {
  constructor(private readonly mainService: MainService) {}

  async onModuleInit() {
    const containers = await this.mainService.listAllServices();
    console.dir(containers, { depth: null, colors: true });
  }
}
