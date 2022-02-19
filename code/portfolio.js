import PropTypes from 'prop-types';
import React, { Fragment } from 'react';


/**
 * The Portfolio component
 *
 * @disable-docs
 */
const Portfolio = ({ _relativeURL, _ID, _nav, _pages }) => (
	<Fragment>
		<div className="section works">
			<div className="content">
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
				<div className="box-items portfolio-items">
					{
						Object.keys(_nav["index"]["portfolio"])
							.map((id, i) => _pages[id])
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

Portfolio.propTypes = {
	/**
	 * _body: (test)(12)
	 */
	_body: PropTypes.node.isRequired,
};

Portfolio.defaultProps = {};

export default Portfolio;
