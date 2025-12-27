from .base import BaseCommand
from ..utils import ServiceRunner


class RepoCommand(BaseCommand):
    @property
    def name(self):
        return "repo"

    @property
    def help(self):
        return "Manages the git repository (status, branch, update, reset)."

    def register_args(self, parser):
        # We use a sub-parser 'action' to mimic: km repo <action> [args]
        subparsers = parser.add_subparsers(
            dest="action", required=True, help="Repo actions"
        )

        # 1. Status
        subparsers.add_parser("status", help="Show git status and branch info")

        # 2. Update
        subparsers.add_parser("update", help="Pull latest changes")

        # 3. Reset
        subparsers.add_parser("reset", help="Hard reset to HEAD")

        # 4. Branch
        branch_parser = subparsers.add_parser(
            "branch", help="Checkout or create a branch/tag"
        )
        branch_parser.add_argument("name", help="Name of the branch or tag")

        # 5. Delete
        del_parser = subparsers.add_parser("delete", help="Delete a local branch")
        del_parser.add_argument("name", help="Name of the branch to delete")

    def run(self, args):
        # Dispatch to RepoService.sh
        # The bash script expects: $1=action, $2=arg

        service_args = [args.action]

        if args.action in ["branch", "delete"]:
            service_args.append(args.name)

        ServiceRunner.run("RepoService.sh", service_args)
