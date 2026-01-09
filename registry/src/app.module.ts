import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { DockerModule } from './docker/docker.module';
import { ConfigModule } from '@nestjs/config';
import { MainModule } from './main/main.module';
import { OpenapiModule } from './openapi/openapi.module';
import { GateModule } from './gate/gate.module';

@Module({
  imports: [
    ConfigModule.forRoot({ ignoreEnvFile: true, isGlobal: true }),
    DockerModule,
    MainModule,
    OpenapiModule,
    GateModule,
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
