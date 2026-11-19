from ssz.uint import BaseUint as Uint
from ssz.boolean import Boolean
from ssz.container import Container, ProgressiveContainer
from ssz.collections import List, Vector, ProgressiveList
from eth_consensus_specs.debug.decode import decode
import re


def normalize(value, typ):
    if issubclass(typ, Boolean):
        if isinstance(value, bool):
            return value
        assert value in ("true", "false")
        return value == "true"
    if issubclass(typ, Uint):
        if isinstance(value, str):
            assert re.fullmatch("[0-9]+", value)
            return int(value)
        return value
    if issubclass(typ, Container | ProgressiveContainer):
        return {
            name: normalize(value[name], field.annotation)
            for name, field in typ.model_fields.items()
        }
    if isinstance(value, list) and issubclass(typ, List | Vector | ProgressiveList):
        return [normalize(item, typ.ELEMENT_TYPE) for item in value]
    return value


def parsed(raw, typ):
    return decode(normalize(raw, typ), typ)
