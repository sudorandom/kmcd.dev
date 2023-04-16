import PropTypes from 'prop-types';
import React, { Fragment } from 'react';


/**
 * The Blog component
 *
 * @disable-docs
 */
const Blog = ({ preface, _relativeURL, _ID, _nav, _pages }) => (
	<Fragment>
		<div className="section works">
			<div className="content">
				<p>
					{ preface }
				</p>
				<div className="filter-menu">
					<div className="filters">
						<div className="btn-group">
							<label data-text="All" className="glitch-effect">
								<input type="radio" name="fl_radio" value=".box-item" />All
							</label>
						</div>
						<div className="btn-group">
							<label data-text="Gaming">
								<input type="radio" name="fl_radio" value=".f-gaming" />Gaming
							</label>
						</div>
						<div className="btn-group">
							<label data-text="Data Visualization">
								<input type="radio" name="fl_radio" value=".f-data-visualization" />Data Visualization
							</label>
						</div>
					</div>
				</div>
				<div className="box-items blog-items">
					{
						Object.keys(_nav["index"]["blog"])
							.map((id, i) => _pages[id]).sort(function(a, b) {
								if (a.date > b.date) {
									return -1
								} else if (a.date > b.date) {
									return 1
								} else {
									return 0
								}
							})
							.map(
								(page, i) =>(
									<div key={i} className={`box-item ${page.categories.map((cat) => "f-"+cat).join(" ")}`}>
										<div className="image">
											<a href={page._url}>
												<img src={_relativeURL( page.thumbnail, _ID )} alt="" />
												<span className="info">
													<span className="centrize full-width">
														<span className="vertical-center">
															<span className="ion ion-code"></span>
														</span>
													</span>
												</span>
											</a>
										</div>
										<div className="desc">
											<div className="date">{page.date}</div>
											<a href={page._url} className="name has-popup-link">{page.title}</a>
										</div>
									</div>
								)
							)
					}
				</div>
				<div className="clear"></div>
			</div>
		</div>
	</Fragment>
);

Blog.propTypes = {
	/**
	 * _body: (test)(12)
	 */
	_body: PropTypes.node.isRequired,
};

Blog.defaultProps = {};

export default Blog;
