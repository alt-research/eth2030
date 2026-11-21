# Snap historical resume retention boundary

The selected EIP-7928 revision 3db38a3725105cb67b8804ac3e147967113bc3ca requires at least14299 epochs of BAL serving and explicitly permits pruned historical BAL bytes to be null. A saved snap body/BAL prefix must keep canonical hash/header/body authentication after this retention window expires; missing retained bytes and corrupted bytes still fail. The current cursor validator requires every old BAL and cannot resume after legal expiry. A dedicated correction and boundary controls are in progress.
