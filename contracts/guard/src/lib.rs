use near_sdk::{env, near, require, AccountId, PanicOnDefault};

/// Policy storage only. Does not custody funds or authorize an external executor.
#[near(contract_state)]
#[derive(PanicOnDefault)]
pub struct Guard {
    owner: AccountId,
    paused: bool,
    trigger_bps: u32,
    target_bps: u32,
}

#[near]
impl Guard {
    #[init]
    pub fn new() -> Self {
        Self {
            owner: env::predecessor_account_id(),
            paused: true,
            trigger_bps: 12_000,
            target_bps: 15_000,
        }
    }

    pub fn set_policy(&mut self, trigger_bps: u32, target_bps: u32) {
        self.assert_owner();
        require!(
            trigger_bps > 10_000,
            "Trigger must exceed liquidation boundary"
        );
        require!(target_bps >= trigger_bps, "Target must be at least trigger");
        self.trigger_bps = trigger_bps;
        self.target_bps = target_bps;
    }

    pub fn set_paused(&mut self, paused: bool) {
        self.assert_owner();
        self.paused = paused;
    }

    pub fn get_policy(&self) -> (u32, u32, bool) {
        (self.trigger_bps, self.target_bps, self.paused)
    }

    fn assert_owner(&self) {
        require!(env::predecessor_account_id() == self.owner, "Owner only");
    }
}
