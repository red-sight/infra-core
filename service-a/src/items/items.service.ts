import { Injectable, NotFoundException } from '@nestjs/common';
import { Item } from './item.entity';
import { CreateItemDto } from './dto/create-item.dto';

@Injectable()
export class ItemsService {
  private readonly items: Item[] = [];
  private nextId = 1;

  findAll(): Item[] {
    return this.items;
  }

  findOne(id: number): Item {
    const item = this.items.find((i) => i.id === id);
    if (!item) throw new NotFoundException(`Item ${id} not found`);
    return item;
  }

  create(dto: CreateItemDto): Item {
    const item: Item = { id: this.nextId++, ...dto };
    this.items.push(item);
    return item;
  }

  remove(id: number): void {
    const index = this.items.findIndex((i) => i.id === id);
    if (index === -1) throw new NotFoundException(`Item ${id} not found`);
    this.items.splice(index, 1);
  }
}
