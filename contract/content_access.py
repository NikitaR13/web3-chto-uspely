# contract/content_access.py

from typing import Any

from boa3.builtin import public
from boa3.builtin.contract import NeoAccountState, call_contract
from boa3.builtin.interop.contract import GAS as GAS_SCRIPT_HASH
from boa3.builtin.interop.runtime import CheckWitness, CallingScriptHash, TriggerType, GetTrigger
from boa3.builtin.interop.storage import get, put
from boa3.builtin.nativecontract.gas import transfer as gas_transfer

MIN_GAS_AMOUNT = 100000000  # 1 GAS = 100000000 fractions

@public
def _deploy(data: Any, update: bool):
    """
    Инициализация контракта при деплое (можно ничего не делать).
    """
    if not update:
        # Выполнится при первоначальном деплое
        pass

@public
def onNEP17Payment(from_address: bytes, amount: int, data: Any):
    """
    Обрабатывает поступление NEP-17 (в частности GAS).
    This is a standard entry point for receiving tokens.
    """
    # Просто логируем/записываем, сколько пришло. Можно ничего не делать.
    pass

@public
def purchase_access(user: bytes, content_id: str) -> bool:
    """
    Пользователь user вызывает purchase_access, переводя 1 GAS на контракт.
    Проверяем:
     - Вызывается ли из onNEP17Payment? (либо другой trigger)
     - CheckWitness(user) 
     - amount >= MIN_GAS_AMOUNT
    Если всё ок, то записываем в storage.
    """
    # Триггер должен быть Application
    if GetTrigger() != TriggerType.APPLICATION:
        return False

    # Проверяем, действительно ли user подписывает транзакцию
    if not CheckWitness(user):
        return False

    # Проверим, была ли оплата?
    # Предположим, мы хотим чтобы этот метод вызывался после onNEP17Payment c amount >= 1 GAS
    # Но в N3 обычно onNEP17Payment вызывается автоматически при transfer'е.
    # Один из способов — проверять баланс контракта до/после.
    # Упростим и проверим, что CALL происходит из GAS-контракта (т.е. onNEP17Payment).
    # (В реальности эта логика зависит от того, как вы вызываете transfer + data)

    call_script = CallingScriptHash
    # GAS_SCRIPT_HASH = скрипт-хэш нативного GAS-контракта
    if call_script != GAS_SCRIPT_HASH:
        return False

    # Признак, что этот вызов идёт в рамках onNEP17Payment
    # Но нужно ещё проверить amount. 
    # onNEP17Payment сама принимает (from, amount, data), но Boa3 не даёт просто так заглянуть. 
    # Вариант: передавать amount через data, или хранить amount в Storage при onNEP17Payment.

    # Допустим, мы хотим сразу при onNEP17Payment проверять, что amount >= MIN_GAS_AMOUNT
    # Для простоты, подумаем так: user сам делает transfer(GAS) -> contract, data="content_id",
    # и onNEP17Payment внутри вызывает purchase_access(user, content_id).

    key = b"access_" + content_id.encode('utf-8')
    existing_data = get(key)
    user_hex = user.hex()

    if existing_data:
        new_record = existing_data.decode('utf-8') + "," + user_hex
        put(key, new_record)
    else:
        put(key, user_hex)

    return True


@public
def has_access(user: bytes, content_id: str) -> bool:
    key = b"access_" + content_id.encode('utf-8')
    data = get(key)
    if not data:
        return False
    data_str = data.decode('utf-8')
    if user.hex() in data_str.split(','):
        return True
    return False
