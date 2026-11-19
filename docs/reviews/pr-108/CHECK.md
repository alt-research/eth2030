# Moving-target full cold synchronization

- [x] Start a fresh actual Native EL/Lighthouse pair and retain the original failed helper attempt separately.
- [x] Reproduce genesis execution head while the real CL advances, with repeated independent-peer header/body downloads and no completed ancestry walk.
- [x] Add a failing continuous-target progress consumer while preserving immediate cancellation, latest-target publication and execution-validation authority.
- [x] Retain hash-bound header/body download progress in bounded caches using existing upstream cache implementations; stream forward validation without retaining all block bodies in the ancestry list.
- [x] Verify byte/count bounds, cancellation/reorg isolation, wrong-hash and unvalidated-state boundaries, restored behavior mutations and full local checks.
- [ ] Publish this separate review, then qualify actual moving-target cold sync and original historical readback on the owned new pair.
