package gridmap

import (
	"rx1/foundation"
)

type Tile struct {
	Feature            foundation.TileType // we need this
	DefinedDescription string
	IsWalkable         bool // this
	IsTransparent      bool // this
}

func (t Tile) Icon() foundation.TileType {
	return t.Feature
}

func (t Tile) Description() string {
	return t.DefinedDescription
}

func (t Tile) IsTree() bool {
	return t.Feature == foundation.TileTree
}

func (t Tile) IsWater() bool {
	return t.Feature == foundation.TileWater
}

func (t Tile) IsMountain() bool {
	return t.Feature == foundation.TileMountain || t.Feature == foundation.TileMountainPeak
}

func (t Tile) IsVoid() bool {
	return t.Feature == foundation.TileEmpty
}

func (t Tile) IsStairsUp() bool {
	return t.Feature == foundation.TileStairsUp || t.Feature == foundation.TileCaveStairsUp
}

func (t Tile) IsStairsDown() bool {
	return t.Feature == foundation.TileStairsDown || t.Feature == foundation.TileTownStairsDown || t.Feature == foundation.TileCaveStairsDown
}

func (t Tile) IsChasm() bool {
	return t.Feature == foundation.TileChasm
}

func (t Tile) IsStairs() bool {
	return t.IsStairsUp() || t.IsStairsDown()
}

func (t Tile) IsLava() bool {
	return t.Feature == foundation.TileLava
}

func (t Tile) IsSpecial() bool {
	return t.IsStairs() || t.IsChasm() || t.IsLava() || t.IsVoid() || t.IsWater() || t.IsMountain() || t.IsTree() || t.IsDoor()
}

func (t Tile) IsDoor() bool {
	return t.Feature == foundation.TileDoorClosed || t.Feature == foundation.TileDoorOpen
}

func (t Tile) IsVendor() bool {
	return t.Feature == foundation.TileVendorCurator || t.Feature == foundation.TileVendorBlacksmith || t.Feature == foundation.TileVendorGeneral || t.Feature == foundation.TileVendorHome
}

type MapCell[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}] struct {
	TileType   Tile
	IsExplored bool
	IsLit      bool // IsLit is true if this tile receives light from a light source and is thus permanently lit if it's explored.
	Actor      *ActorType
	Item       *ItemType
	Object     *ObjectType
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithItemHereRemoved(itemHere ItemType) MapCell[ActorType, ItemType, ObjectType] {
	if c.Item != nil && *c.Item == itemHere {
		c.Item = nil
	}
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithObjectHereRemoved(obj ObjectType) MapCell[ActorType, ItemType, ObjectType] {
	if c.Object != nil && *c.Object == obj {
		c.Object = nil
	}
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithActor(actor ActorType) MapCell[ActorType, ItemType, ObjectType] {
	c.Actor = &actor
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithObject(obj ObjectType) MapCell[ActorType, ItemType, ObjectType] {
	c.Object = &obj
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithActorHereRemoved(actorHere ActorType) MapCell[ActorType, ItemType, ObjectType] {
	if c.Actor != nil && *c.Actor == actorHere {
		c.Actor = nil
	}
	return c
}

func (c MapCell[ActorType, ItemType, ObjectType]) WithItem(item ItemType) MapCell[ActorType, ItemType, ObjectType] {
	c.Item = &item
	return c
}
