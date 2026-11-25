# Roadmap scope explanation display

Remove the Official upgrade scope explanation card from the roadmap page.
Keep the official scope data, chart computation/rendering, homepage explanation
and collapsed progress sections intact. Update the existing export checks to
match the page-specific display requirement.

Validation passes: site production build, complete exported-site checks and the
149-entry EIP tracker audit (422 referenced package paths, none missing).
The complete exported six-card chart section is byte-identical to the master
site source baseline (`f04bc4395d1938119655f30b592d15baf3bedd4b`). The roadmap
scope card is absent and the homepage explanation remains present.
