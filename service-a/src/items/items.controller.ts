import {
  Controller,
  Get,
  Post,
  Delete,
  Param,
  Body,
  ParseIntPipe,
  HttpCode,
} from '@nestjs/common';
import {
  ApiTags,
  ApiOperation,
  ApiResponse,
  ApiBearerAuth,
  ApiExtension,
} from '@nestjs/swagger';
import { ItemsService } from './items.service';
import { CreateItemDto } from './dto/create-item.dto';
import { Item } from './item.entity';

@ApiTags('items')
@Controller('items')
export class ItemsController {
  constructor(private readonly itemsService: ItemsService) {}

  @Get()
  @ApiOperation({ summary: 'List all items' })
  @ApiResponse({ status: 200, type: [Item] })
  @ApiExtension('x-infra-protected', false)
  findAll(): Item[] {
    return this.itemsService.findAll();
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get item by ID' })
  @ApiResponse({ status: 200, type: Item })
  @ApiResponse({ status: 404, description: 'Not found' })
  @ApiExtension('x-infra-protected', false)
  findOne(@Param('id', ParseIntPipe) id: number): Item {
    return this.itemsService.findOne(id);
  }

  @Post()
  @ApiBearerAuth('bearer')
  @ApiOperation({ summary: 'Create item' })
  @ApiResponse({ status: 201, type: Item })
  @ApiExtension('x-infra-protected', true)
  @ApiExtension('x-infra-scopes', ['write:items'])
  create(@Body() dto: CreateItemDto): Item {
    return this.itemsService.create(dto);
  }

  @Delete(':id')
  @HttpCode(204)
  @ApiBearerAuth('bearer')
  @ApiOperation({ summary: 'Delete item' })
  @ApiResponse({ status: 204 })
  @ApiResponse({ status: 404, description: 'Not found' })
  @ApiExtension('x-infra-protected', true)
  @ApiExtension('x-infra-scopes', ['write:items'])
  remove(@Param('id', ParseIntPipe) id: number): void {
    this.itemsService.remove(id);
  }
}
