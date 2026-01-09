import { Injectable } from '@nestjs/common';
import { ServiceInfo } from '../types';
import axios from 'axios';
import { AxiosError } from 'axios';
import { Swagger } from '@atlassian/atlassian-openapi';

@Injectable()
export class OpenapiService {
  async fetchSpec(service: ServiceInfo) {
    try {
      const res = await axios<Swagger.SwaggerV3>({
        url: `http://${service.ip}:${service.port}/${service.oasEndpoint}`,
        timeout: 3000,
      });
      return res.data;
    } catch (e) {
      if (e instanceof AxiosError) {
        console.error(e.toJSON());
      } else {
        console.error(e);
      }
    }
  }
}
