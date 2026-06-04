import {
  Controller,
  Get,
  Post,
  Delete,
  Param,
  Body,
  ParseIntPipe,
  HttpCode,
} from "@nestjs/common";
import { ApiTags, ApiOperation, ApiResponse } from "@nestjs/swagger";
import { InfraAuth, InfraPublic } from "../infra";
import { ItemsService } from "./items.service";
import { CreateItemDto } from "./dto/create-item.dto";
import { Item } from "./item.entity";

@ApiTags("items")
@Controller("items")
export class ItemsController {
  constructor(private readonly itemsService: ItemsService) {}

  @Get()
  @InfraPublic()
  @ApiOperation({ summary: "List all items" })
  @ApiResponse({ status: 200, type: [Item] })
  findAll(): Item[] {
    return this.itemsService.findAll();
  }

  @Get(":id")
  @InfraPublic()
  @ApiOperation({ summary: "Get item by ID" })
  @ApiResponse({ status: 200, type: Item })
  @ApiResponse({ status: 404, description: "Not found" })
  findOne(@Param("id", ParseIntPipe) id: number): Item {
    return this.itemsService.findOne(id);
  }

  @Post()
  @InfraAuth("write:items")
  @ApiOperation({ summary: "Create item" })
  @ApiResponse({ status: 201, type: Item })
  create(@Body() dto: CreateItemDto): Item {
    return this.itemsService.create(dto);
  }

  @Delete(":id")
  @HttpCode(204)
  @InfraAuth("delete:items")
  @ApiOperation({ summary: "Delete item" })
  @ApiResponse({ status: 204 })
  @ApiResponse({ status: 404, description: "Not found" })
  remove(@Param("id", ParseIntPipe) id: number): void {
    this.itemsService.remove(id);
  }
}
