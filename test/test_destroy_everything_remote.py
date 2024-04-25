# import logging
#
# logger = logging.getLogger(__name__)
#
#
# class TestDestroyEverythingRemote:
#     """
#     If something drastic has failed during terraform and teardown hasn't occurred properly, use this test to force
#     a teardown of our infrastructure to occur.
#
#     e.g. passing the following into pytest after uncommenting this file:
#     test_destroy_everything_remote.py::TestDestroyEverythingRemote::test_delete_all
#     """
#     def test_delete_all(self, destroy_all_remote_infrastructure):
#         logger.info("Deleting everything remote")
#         assert True
