import { Injectable } from "@nestjs/common";
import { ServiceInfo } from "../types";
import axios from "axios";
import { AxiosError } from "axios";

@Injectable()
export class OpenapiService {
  async fetchSpec(service: ServiceInfo) {
    console.log("service", service);
    try {
      const res = await axios({
        url: `http://${service.ip}:${service.port}/${service.oasEndpoint}`,
        timeout: 3000,
      });
      console.dir(res.data, { depth: null, colors: true });
    } catch (e) {
      if (e instanceof AxiosError) {
        console.error(e.toJSON());
      } else {
        console.error(e);
      }
    }

    // console.log("OpenApi spec", res.data);
  }
}
